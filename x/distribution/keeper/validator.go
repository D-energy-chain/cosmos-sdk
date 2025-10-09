package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// initialize rewards for a new validator
func (k Keeper) initializeValidator(ctx context.Context, val stakingtypes.ValidatorI) error {
	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}
	// set initial historical rewards (period 0) with reference count of 1
	// Initialize both native and NFT cumulative reward ratios to empty
	err = k.SetValidatorHistoricalRewards(ctx, valBz, 0, types.NewValidatorHistoricalRewardsWithNFT(sdk.DecCoins{}, sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	// set current rewards (starting at period 1)
	err = k.SetValidatorCurrentRewards(ctx, valBz, types.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	// set accumulated commission
	err = k.SetValidatorAccumulatedCommission(ctx, valBz, types.InitialValidatorAccumulatedCommission())
	if err != nil {
		return err
	}

	// set outstanding rewards
	err = k.SetValidatorOutstandingRewards(ctx, valBz, types.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}})
	return err
}

// increment validator period, returning the period just ended
func (k Keeper) IncrementValidatorPeriod(ctx context.Context, val stakingtypes.ValidatorI) (uint64, error) {
	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return 0, err
	}

	// fetch current rewards
	rewards, err := k.GetValidatorCurrentRewards(ctx, valBz)
	if err != nil {
		return 0, err
	}

	// calculate current reward ratios for both NFT and native delegations
	var current, nftCurrent sdk.DecCoins

	// Try both possible method names for NFT shares
	var nftShares math.LegacyDec

	// First try GetDelegatorNftShares (as per documentation)
	if v, ok := val.(interface{ GetDelegatorNftShares() math.LegacyDec }); ok {
		nftShares = v.GetDelegatorNftShares()
	} else if v, ok := val.(interface{ GetNFTDelegatorShares() math.LegacyDec }); ok {
		// Fallback to GetNFTDelegatorShares
		nftShares = v.GetNFTDelegatorShares()
	} else {
		// Neither method exists
		nftShares = math.LegacyZeroDec()
	}

	nativeShares := val.GetDelegatorShares()
	totalShares := nftShares.Add(nativeShares)

	if val.GetTokens().IsZero() || totalShares.IsZero() {
		// can't calculate ratio for zero-token validators
		// ergo we instead add to the community pool
		feePool, err := k.FeePool.Get(ctx)
		if err != nil {
			return 0, err
		}

		outstanding, err := k.GetValidatorOutstandingRewards(ctx, valBz)
		if err != nil {
			return 0, err
		}

		feePool.CommunityPool = feePool.CommunityPool.Add(rewards.Rewards...)
		outstanding.Rewards = outstanding.GetRewards().Sub(rewards.Rewards)
		err = k.FeePool.Set(ctx, feePool)
		if err != nil {
			return 0, err
		}

		err = k.SetValidatorOutstandingRewards(ctx, valBz, outstanding)
		if err != nil {
			return 0, err
		}

		current = sdk.DecCoins{}
		nftCurrent = sdk.DecCoins{}
	} else {
		// Get the current reward allocation ratios
		nftStakingRatio, err := k.GetNftStakingRatio(ctx)
		if err != nil {
			return 0, err
		}
		nativeStakingRatio, err := k.GetNativeStakingRatio(ctx)
		if err != nil {
			return 0, err
		}

		// LOG: CRITICAL POINT - RE-SPLITTING COMBINED REWARDS
		k.Logger(ctx).Info("PERIOD INCREMENT - RE-SPLITTING REWARDS",
			"validator", val.GetOperator(),
			"===== INPUT (COMBINED DELEGATOR REWARDS) =====", "",
			"current_rewards_total", rewards.Rewards.String(),
			"===== SHARES INFO =====", "",
			"native_shares", nativeShares.String(),
			"nft_shares", nftShares.String(),
			"total_shares", totalShares.String(),
			"===== SPLIT RATIOS =====", "",
			"native_staking_ratio", nativeStakingRatio.String(),
			"nft_staking_ratio", nftStakingRatio.String(),
		)

		// Split the current rewards according to staking ratios
		nftRewards := rewards.Rewards.MulDecTruncate(nftStakingRatio)
		nativeRewards := rewards.Rewards.MulDecTruncate(nativeStakingRatio)

		// LOG: POOL AMOUNTS AFTER RE-SPLIT
		k.Logger(ctx).Info(" RE-SPLIT POOL AMOUNTS",
			"validator", val.GetOperator(),
			"native_pool_amount", nativeRewards.String(),
			"nft_pool_amount", nftRewards.String(),
			"⚠️  ISSUE", "These amounts were calculated by re-splitting combined rewards, not from original allocation",
		)

		// Calculate reward ratios per unit of delegation
		// Native ratio: native rewards / native shares
		if !nativeShares.IsZero() {
			current = nativeRewards.QuoDecTruncate(nativeShares)
		} else {
			current = sdk.DecCoins{}
		}

		// NFT ratio: NFT rewards / NFT shares
		if !nftShares.IsZero() {
			nftCurrent = nftRewards.QuoDecTruncate(nftShares)
		} else {
			nftCurrent = sdk.DecCoins{}
		}

		// LOG: PER-SHARE RATIOS
		k.Logger(ctx).Info(" PER-SHARE REWARD RATIOS",
			"validator", val.GetOperator(),
			"native_ratio_per_share", current.String(),
			"nft_ratio_per_share", nftCurrent.String(),
		)
	}

	// fetch historical rewards for last period
	historical, err := k.GetValidatorHistoricalRewards(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	cumRewardRatio := historical.CumulativeRewardRatio
	nftCumRewardRatio := historical.NftCumulativeRewardRatio

	// decrement reference count
	err = k.decrementReferenceCount(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical rewards with separate cumulative ratios and reference count of 1
	newNativeCumRatio := cumRewardRatio.Add(current...)
	newNftCumRatio := nftCumRewardRatio.Add(nftCurrent...)

	// LOG: FINAL CUMULATIVE RATIOS
	k.Logger(ctx).Info(" CUMULATIVE REWARD RATIOS UPDATED",
		"validator", val.GetOperator(),
		"period", rewards.Period,
		"===== PREVIOUS CUMULATIVE RATIOS =====", "",
		"prev_native_cumulative_ratio", cumRewardRatio.String(),
		"prev_nft_cumulative_ratio", nftCumRewardRatio.String(),
		"===== ADDED THIS PERIOD =====", "",
		"native_ratio_increment", current.String(),
		"nft_ratio_increment", nftCurrent.String(),
		"===== NEW CUMULATIVE RATIOS =====", "",
		"new_native_cumulative_ratio", newNativeCumRatio.String(),
		"new_nft_cumulative_ratio", newNftCumRatio.String(),
		"===== WITHDRAWABLE CALCULATION =====", "",
		"note", "Delegators multiply their stake by (new_ratio - starting_ratio) to get rewards",
	)

	err = k.SetValidatorHistoricalRewards(ctx, valBz, rewards.Period, types.NewValidatorHistoricalRewardsWithNFT(newNativeCumRatio, newNftCumRatio, 1))
	if err != nil {
		return 0, err
	}

	// set current rewards, incrementing period by 1
	err = k.SetValidatorCurrentRewards(ctx, valBz, types.NewValidatorCurrentRewards(sdk.DecCoins{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	// Log period increment
	k.logPeriodIncrement(ctx, sdk.ValAddress(valBz), rewards.Period-1, rewards.Period, "epoch_end")

	return rewards.Period, nil
}

// IncrementAllValidatorPeriods increments periods for all validators
// This is called at epoch end to convert accumulated rewards to cumulative ratios
func (k Keeper) IncrementAllValidatorPeriods(ctx context.Context) error {
	return k.stakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		_, err := k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			k.Logger(ctx).Error("Failed to increment validator period during epoch end", "validator", val.GetOperator(), "error", err)
			// Continue with other validators instead of stopping the entire process
		}
		return false
	})
}

// increment the reference count for a historical rewards value
func (k Keeper) incrementReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.GetValidatorHistoricalRewards(ctx, valAddr, period)
	if err != nil {
		return err
	}
	if historical.ReferenceCount > 2 {
		panic("reference count should never exceed 2")
	}
	historical.ReferenceCount++
	return k.SetValidatorHistoricalRewards(ctx, valAddr, period, historical)
}

// decrement the reference count for a historical rewards value, and delete if zero references remain
func (k Keeper) decrementReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.GetValidatorHistoricalRewards(ctx, valAddr, period)
	if err != nil {
		return err
	}

	if historical.ReferenceCount == 0 {
		panic("cannot set negative reference count")
	}
	historical.ReferenceCount--
	if historical.ReferenceCount == 0 {
		return k.DeleteValidatorHistoricalReward(ctx, valAddr, period)
	}

	return k.SetValidatorHistoricalRewards(ctx, valAddr, period, historical)
}

func (k Keeper) updateValidatorSlashFraction(ctx context.Context, valAddr sdk.ValAddress, fraction math.LegacyDec) error {
	if fraction.GT(math.LegacyOneDec()) || fraction.IsNegative() {
		panic(fmt.Sprintf("fraction must be >=0 and <=1, current fraction: %v", fraction))
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	// increment current period
	newPeriod, err := k.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return err
	}

	// increment reference count on period we need to track
	k.incrementReferenceCount(ctx, valAddr, newPeriod)

	slashEvent := types.NewValidatorSlashEvent(newPeriod, fraction)
	height := uint64(sdkCtx.BlockHeight())

	// Log slash event
	k.logSlashEvent(ctx, valAddr, fraction, newPeriod, height)

	return k.SetValidatorSlashEvent(ctx, valAddr, height, newPeriod, slashEvent)
}
