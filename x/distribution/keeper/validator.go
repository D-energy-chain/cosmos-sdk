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
	err = k.SetValidatorHistoricalRewards(ctx, valBz, 0, types.NewValidatorHistoricalRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}
	err = k.SetValidatorHistoricalNFTRewards(ctx, valBz, 0, types.NewValidatorHistoricalNFTRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	// set current rewards (starting at period 1)
	err = k.SetValidatorCurrentRewards(ctx, valBz, types.NewValidatorCurrentRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return err
	}

	err = k.SetValidatorCurrentNFTRewards(ctx, valBz, types.NewValidatorCurrentNFTRewards(sdk.DecCoins{}, 1))
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
	logger := k.Logger(ctx)
	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return 0, err
	}

	// fetch current rewards
	rewards, err := k.GetValidatorCurrentRewards(ctx, valBz)
	if err != nil {
		return 0, err
	}

	// calculate current ratio
	var current sdk.DecCoins
	if val.GetTokens().IsZero() {

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
	} else {
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		current = rewards.Rewards.QuoDecTruncate(math.LegacyNewDecFromInt(val.GetTokens()))
		logger.Info(
			"validator native reward ratio computation",
			"validator", val.GetOperator(),
			"period", rewards.Period,
			"pending_rewards", rewards.Rewards.String(),
			"validator_tokens", val.GetTokens().String(),
			"computed_ratio_increment", current.String(),
		)
	}

	// fetch historical rewards for last period
	historical, err := k.GetValidatorHistoricalRewards(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	cumRewardRatio := historical.CumulativeRewardRatio

	// decrement reference count
	err = k.decrementReferenceCount(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical rewards with reference count of 1
	err = k.SetValidatorHistoricalRewards(ctx, valBz, rewards.Period, types.NewValidatorHistoricalRewards(cumRewardRatio.Add(current...), 1))
	if err != nil {
		return 0, err
	}
	logger.Info(
		"validator native cumulative reward ratio updated",
		"validator", val.GetOperator(),
		"previous_period", rewards.Period-1,
		"new_period", rewards.Period,
		"previous_cumulative_ratio", cumRewardRatio.String(),
		"current_increment", current.String(),
		"updated_cumulative_ratio", cumRewardRatio.Add(current...).String(),
	)

	// set current rewards, incrementing period by 1
	err = k.SetValidatorCurrentRewards(ctx, valBz, types.NewValidatorCurrentRewards(sdk.DecCoins{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	return rewards.Period, nil
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

// increment validator period, returning the period just ended
func (k Keeper) IncrementValidatorNFTPeriod(ctx context.Context, val stakingtypes.ValidatorI) (uint64, error) {
	logger := k.Logger(ctx)
	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return 0, err
	}

	// fetch current rewards
	rewards, err := k.GetValidatorCurrentNFTRewards(ctx, valBz)
	if err != nil {
		return 0, err
	}

	// a validator with no NFT reward records reads back as period 0; seed them before any
	// arithmetic touches Period-1
	rewards, err = k.ensureValidatorNFTRewardsInitialized(ctx, valBz, rewards)
	if err != nil {
		return 0, err
	}

	// calculate current ratio
	var current sdk.DecCoins
	if val.GetTotalNFTs().IsZero() {

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

		logger.Info(
			"validator nft reward ratio skipped due to zero total NFTs",
			"validator", val.GetOperator(),
			"period", rewards.Period,
			"pending_nft_rewards", rewards.Rewards.String(),
			"total_nfts", val.GetTotalNFTs().String(),
		)

		current = sdk.DecCoins{}
	} else {
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		current = rewards.Rewards.QuoDecTruncate(math.LegacyNewDecFromInt(val.GetTotalNFTs()))
		logger.Info(
			"validator nft reward ratio computation",
			"validator", val.GetOperator(),
			"period", rewards.Period,
			"pending_nft_rewards", rewards.Rewards.String(),
			"validator_tokens", val.GetTotalNFTs().String(),
			"computed_nft_ratio_increment", current.String(),
		)
	}

	// fetch historical rewards for last period
	historical, err := k.GetValidatorHistoricalNFTRewards(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	cumRewardRatio := historical.NftCumulativeRewardRatio

	// decrement reference count
	err = k.decrementNFTReferenceCount(ctx, valBz, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical rewards with reference count of 1
	err = k.SetValidatorHistoricalNFTRewards(ctx, valBz, rewards.Period, types.NewValidatorHistoricalNFTRewards(cumRewardRatio.Add(current...), 1))
	if err != nil {
		return 0, err
	}
	logger.Info(
		"validator nft cumulative reward ratio updated",
		"validator", val.GetOperator(),
		"previous_period", rewards.Period-1,
		"new_period", rewards.Period,
		"previous_nft_cumulative_ratio", cumRewardRatio.String(),
		"nft_ratio_increment", current.String(),
		"updated_nft_cumulative_ratio", cumRewardRatio.Add(current...).String(),
	)

	// set current rewards, incrementing period by 1
	err = k.SetValidatorCurrentNFTRewards(ctx, valBz, types.NewValidatorCurrentNFTRewards(sdk.DecCoins{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	return rewards.Period, nil
}

// ensureValidatorNFTRewardsInitialized backfills the NFT reward records for a validator that
// never received them from initializeValidator. The known case is a chain restored from an
// exported genesis: staking InitGenesis skips AfterValidatorCreated when data.Exported is set,
// so nothing writes these records for pre-existing validators.
//
// A missing record unmarshals to the zero value, so period 0 is the signal - it is never valid
// for an initialized validator, whose current period starts at 1. Without this backfill,
// IncrementValidatorNFTPeriod computes Period-1 on a uint64 zero, wraps to 2^64-1, finds no
// historical record there, and decrementNFTReferenceCount panics with "cannot set negative
// reference count", halting the chain from BeginBlocker.
//
// Rewards already accrued against the uninitialized record are carried over.
func (k Keeper) ensureValidatorNFTRewardsInitialized(
	ctx context.Context, valAddr sdk.ValAddress, rewards types.ValidatorCurrentRewards,
) (types.ValidatorCurrentRewards, error) {
	if rewards.Period != 0 {
		return rewards, nil
	}

	err := k.SetValidatorHistoricalNFTRewards(ctx, valAddr, 0, types.NewValidatorHistoricalNFTRewards(sdk.DecCoins{}, 1))
	if err != nil {
		return rewards, err
	}

	rewards = types.NewValidatorCurrentNFTRewards(rewards.Rewards, 1)
	if err := k.SetValidatorCurrentNFTRewards(ctx, valAddr, rewards); err != nil {
		return rewards, err
	}

	k.Logger(ctx).Info(
		"backfilled missing validator NFT reward records",
		"validator", valAddr.String(),
		"carried_rewards", rewards.Rewards.String(),
	)

	return rewards, nil
}

// increment the reference count for a historical rewards value
func (k Keeper) incrementNFTReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.GetValidatorHistoricalNFTRewards(ctx, valAddr, period)
	if err != nil {
		return err
	}
	if historical.ReferenceCount > 2 {
		panic("reference count should never exceed 2")
	}
	historical.ReferenceCount++
	return k.SetValidatorHistoricalNFTRewards(ctx, valAddr, period, historical)
}

// decrement the reference count for a historical rewards value, and delete if zero references remain
func (k Keeper) decrementNFTReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.GetValidatorHistoricalNFTRewards(ctx, valAddr, period)
	if err != nil {
		return err
	}

	if historical.ReferenceCount == 0 {
		panic("cannot set negative reference count")
	}
	historical.ReferenceCount--
	if historical.ReferenceCount == 0 {
		return k.DeleteValidatorHistoricalNFTReward(ctx, valAddr, period)
	}

	return k.SetValidatorHistoricalNFTRewards(ctx, valAddr, period, historical)
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
