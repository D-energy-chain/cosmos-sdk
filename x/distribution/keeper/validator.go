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

	// Try both possible method names for NFT shares with debugging
	var nftShares math.LegacyDec
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger()

	// First try GetDelegatorNftShares (as per documentation)
	if v, ok := val.(interface{ GetDelegatorNftShares() math.LegacyDec }); ok {
		nftShares = v.GetDelegatorNftShares()
		logger.Info("IncrementValidatorPeriod: Using GetDelegatorNftShares method",
			"validator", val.GetOperator(),
			"nft_shares", nftShares)
	} else if v, ok := val.(interface{ GetNFTDelegatorShares() math.LegacyDec }); ok {
		// Fallback to GetNFTDelegatorShares
		nftShares = v.GetNFTDelegatorShares()
		logger.Info("IncrementValidatorPeriod: Using GetNFTDelegatorShares method",
			"validator", val.GetOperator(),
			"nft_shares", nftShares)
	} else {
		// Neither method exists
		nftShares = math.LegacyZeroDec()
		logger.Warn("IncrementValidatorPeriod: No NFT shares method found on validator",
			"validator", val.GetOperator(),
			"validator_type", fmt.Sprintf("%T", val))
	}

	nativeShares := val.GetDelegatorShares()
	totalShares := nftShares.Add(nativeShares)

	logger.Info("IncrementValidatorPeriod: Share information",
		"validator", val.GetOperator(),
		"nft_shares", nftShares,
		"native_shares", nativeShares,
		"total_shares", totalShares,
		"validator_tokens", val.GetTokens())

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

		// Split the current rewards according to staking ratios
		nftRewards := rewards.Rewards.MulDecTruncate(nftStakingRatio)
		nativeRewards := rewards.Rewards.MulDecTruncate(nativeStakingRatio)

		logger.Info("IncrementValidatorPeriod: Reward splitting",
			"validator", val.GetOperator(),
			"total_rewards", rewards.Rewards,
			"nft_staking_ratio", nftStakingRatio,
			"native_staking_ratio", nativeStakingRatio,
			"nft_rewards", nftRewards,
			"native_rewards", nativeRewards)

		// Calculate reward ratios per unit of delegation
		// Native ratio: native rewards / native shares
		if !nativeShares.IsZero() {
			current = nativeRewards.QuoDecTruncate(nativeShares)
			logger.Info("IncrementValidatorPeriod: Native ratio calculated",
				"validator", val.GetOperator(),
				"native_rewards", nativeRewards,
				"native_shares", nativeShares,
				"native_ratio", current)
		} else {
			current = sdk.DecCoins{}
			logger.Info("IncrementValidatorPeriod: No native shares, zero native ratio",
				"validator", val.GetOperator())
		}

		// NFT ratio: NFT rewards / NFT shares
		if !nftShares.IsZero() {
			nftCurrent = nftRewards.QuoDecTruncate(nftShares)
			logger.Info("IncrementValidatorPeriod: NFT ratio calculated",
				"validator", val.GetOperator(),
				"nft_rewards", nftRewards,
				"nft_shares", nftShares,
				"nft_ratio", nftCurrent)
		} else {
			nftCurrent = sdk.DecCoins{}
			logger.Info("IncrementValidatorPeriod: No NFT shares, zero NFT ratio",
				"validator", val.GetOperator())
		}
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

	logger.Info("IncrementValidatorPeriod: Updating historical rewards",
		"validator", val.GetOperator(),
		"period", rewards.Period,
		"old_native_cum_ratio", cumRewardRatio,
		"old_nft_cum_ratio", nftCumRewardRatio,
		"current_native_ratio", current,
		"current_nft_ratio", nftCurrent,
		"new_native_cum_ratio", newNativeCumRatio,
		"new_nft_cum_ratio", newNftCumRatio)

	err = k.SetValidatorHistoricalRewards(ctx, valBz, rewards.Period, types.NewValidatorHistoricalRewardsWithNFT(newNativeCumRatio, newNftCumRatio, 1))
	if err != nil {
		return 0, err
	}

	// set current rewards, incrementing period by 1
	err = k.SetValidatorCurrentRewards(ctx, valBz, types.NewValidatorCurrentRewards(sdk.DecCoins{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	return rewards.Period, nil
}

// IncrementAllValidatorPeriods increments periods for all validators
// This is called at epoch end to convert accumulated rewards to cumulative ratios
func (k Keeper) IncrementAllValidatorPeriods(ctx context.Context) error {
	return k.stakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		_, err := k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Error("Failed to increment validator period during epoch end",
				"validator", val.GetOperator(),
				"error", err)
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

	return k.SetValidatorSlashEvent(ctx, valAddr, height, newPeriod, slashEvent)
}
