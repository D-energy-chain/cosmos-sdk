package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// resetDelegatorStartingInfoToPeriod resets the delegator's starting info to a specific period
// after rewards have been distributed. This is used after automatic distribution to set the
// starting period to the ending period that was just used for calculation.
func (k Keeper) resetNFTDelegatorStartingInfoToPeriod(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress, period uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get validator for token calculations
	validator, err := k.stakingKeeper.Validator(ctx, val)
	if err != nil {
		return err
	}

	// Try to get native delegation
	var stake math.LegacyDec = math.LegacyZeroDec()
	delegation, err := k.stakingKeeper.Delegation(ctx, del, val)
	if err == nil && delegation != nil {
		stake = validator.TokensFromSharesTruncated(delegation.GetShares())
	}

	// Try to get NFT delegation
	var nftStake math.LegacyDec = math.LegacyZeroDec()
	nftShares, err := k.stakingKeeper.GetNFTDelegatorShares(ctx, del, val)
	if err == nil {
		nftStake = nftShares
	}

	// Set starting info with the specified period
	startingInfo := types.NewDelegatorStartingInfoWithNFT(period, stake, nftStake, uint64(sdkCtx.BlockHeight()))

	k.Logger(ctx).Debug("Reset delegator starting info",
		"delegator", del.String(),
		"validator", validator.GetOperator(),
		"new_starting_period", period,
		"native_stake", stake.String(),
		"nft_stake", nftStake.String(),
	)

	return k.SetDelegatorStartingInfo(ctx, val, del, startingInfo)
}

// initialize starting info for a new delegation
func (k Keeper) initializeNFTDelegation(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress) error {
	// period has already been incremented - we want to store the period ended by this delegation action
	valNFTCurrentRewards, err := k.GetValidatorCurrentNFTRewards(ctx, val)
	if err != nil {
		return err
	}
	previousPeriod := valNFTCurrentRewards.Period - 1

	// Check if starting info already exists - if so, don't initialize again
	hasInfo, err := k.HasNFTDelegatorStartingInfo(ctx, val, del)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	k.Logger(ctx).Info("initializeDelegation context",
		"block_height", sdkCtx.BlockHeight(),
	)

	if hasInfo {
		// Reset starting info on any share change (native or NFT):
		// - Lock in the current previous period
		// - Refresh height to current block for accurate pro-rating

		// Fetch existing starting info to manage reference counts
		existingInfo, err := k.GetDelegatorStartingInfo(ctx, val, del)
		if err != nil {
			return err
		}

		// If there is no stake change (neither native nor NFT), keep as-is
		var newNftStake math.LegacyDec = math.LegacyZeroDec()
		if nftShares, err := k.stakingKeeper.GetNFTDelegatorShares(ctx, del, val); err == nil {
			newNftStake = nftShares
		}

		// Also recompute native stake
		validator, err := k.stakingKeeper.Validator(ctx, val)
		if err != nil {
			return err
		}

		var newNativeStake math.LegacyDec = math.LegacyZeroDec()
		if delegation, err := k.stakingKeeper.Delegation(ctx, del, val); err == nil && delegation != nil {
			newNativeStake = validator.TokensFromSharesTruncated(delegation.GetShares())
		}

		if newNativeStake.Equal(existingInfo.Stake) && newNftStake.Equal(existingInfo.NftStake) && existingInfo.Height != 0 {
			return nil
		}

		// Decrement reference on the previous starting period
		if err := k.decrementReferenceCount(ctx, val, existingInfo.PreviousPeriod); err != nil {
			return err
		}

		// Increment reference for the new previousPeriod (currentRewards.Period - 1)
		if err := k.incrementReferenceCount(ctx, val, previousPeriod); err != nil {
			return err
		}

		// Write fresh starting info with current block height. Preserve native height if native stake unchanged.
		newInfo := types.NewDelegatorStartingInfoWithNFT(previousPeriod, newNativeStake, newNftStake, uint64(sdkCtx.BlockHeight()))
		if newNativeStake.Equal(existingInfo.Stake) && existingInfo.Height != 0 {
			newInfo.Height = existingInfo.Height
		}
		if !newNftStake.IsZero() && newInfo.NftHeight == 0 {
			newInfo.NftHeight = uint64(sdkCtx.BlockHeight())
		}
		k.Logger(ctx).Info("Reset delegator starting info on share change",
			"delegator", del.String(),
			"validator", validator.GetOperator(),
			"old_previous_period", existingInfo.PreviousPeriod,
			"new_previous_period", previousPeriod,
			"height_set", newInfo.Height,
			"native_stake", newNativeStake.String(),
			"nft_stake", newNftStake.String(),
		)
		return k.SetDelegatorStartingInfo(ctx, val, del, newInfo)
	}

	// Create new starting info

	// increment reference count for the period we're going to track
	err = k.incrementReferenceCount(ctx, val, previousPeriod)
	if err != nil {
		return err
	}

	validator, err := k.stakingKeeper.Validator(ctx, val)
	if err != nil {
		return err
	}

	// Try to get native delegation - it's okay if it doesn't exist for NFT-only delegators
	var stake math.LegacyDec = math.LegacyZeroDec()
	delegation, err := k.stakingKeeper.Delegation(ctx, del, val)
	if err == nil && delegation != nil {
		// calculate delegation stake in tokens
		// we don't store directly, so multiply delegation shares * (tokens per share)
		// note: necessary to truncate so we don't allow withdrawing more rewards than owed
		stake = validator.TokensFromSharesTruncated(delegation.GetShares())
	}

	// calculate NFT delegation stake (if any)
	var nftStake math.LegacyDec = math.LegacyZeroDec()
	nftShares, nftSharesErr := k.stakingKeeper.GetNFTDelegatorShares(ctx, del, val)

	if nftSharesErr == nil {
		// NFT shares represent the delegator's proportion of NFT delegations to this validator
		nftStake = nftShares
	}

	// Ensure at least one type of delegation exists
	if stake.IsZero() && nftStake.IsZero() {
		return types.ErrNoDelegationExists
	}

	startingInfo := types.NewDelegatorStartingInfoWithNFT(previousPeriod, stake, nftStake, uint64(sdkCtx.BlockHeight()))
	// If there is NFT stake, ensure nft_height is set; otherwise it defaults to native height
	if !nftStake.IsZero() && startingInfo.NftHeight == 0 {
		startingInfo.NftHeight = uint64(sdkCtx.BlockHeight())
	}
	k.Logger(ctx).Info("Setting DelegatorStartingInfo",
		"delegator", del.String(),
		"validator", validator.GetOperator(),
		"previous_period", previousPeriod,
		"height_set", startingInfo.Height,
	)
	return k.SetDelegatorStartingInfo(ctx, val, del, startingInfo)
}

// calculate the rewards accrued by an NFT delegation between two periods
func (k Keeper) calculateNFTDelegationRewardsBetween(ctx context.Context, val stakingtypes.ValidatorI,
	startingPeriod, endingPeriod uint64, nftStake math.LegacyDec,
) (sdk.DecCoins, error) {
	// sanity check
	if startingPeriod > endingPeriod {
		panic("startingPeriod cannot be greater than endingPeriod")
	}

	// sanity check
	if nftStake.IsNegative() {
		panic("nft stake should not be negative")
	}

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		panic(err)
	}

	// Period 0 is a logical baseline (zero cumulative rewards), not a database entry
	// This eliminates the need to store period 0 for every validator
	var starting types.ValidatorHistoricalNFTRewards
	if startingPeriod == 0 {
		// Period 0 represents genesis/baseline: zero cumulative rewards
		// Use empty slice explicitly, not sdk.NewDecCoins() which returns nil with no args
		starting = types.ValidatorHistoricalNFTRewards{
			NftCumulativeRewardRatio: sdk.DecCoins{},
			ReferenceCount:           0,
			Height:                   0,
		}
	} else {
		// Fetch actual historical rewards for non-zero periods
		starting, err = k.GetValidatorHistoricalNFTRewards(ctx, valBz, startingPeriod)
		if err != nil {
			k.Logger(ctx).Error("Missing starting period historical rewards (NFT)",
				"validator", val.GetOperator(),
				"period", startingPeriod,
				"error", err.Error(),
			)
			return sdk.DecCoins{}, err
		}
	}

	ending, err := k.GetValidatorHistoricalNFTRewards(ctx, valBz, endingPeriod)
	if err != nil {
		k.Logger(ctx).Error("Missing ending period historical rewards (NFT)",
			"validator", val.GetOperator(),
			"period", endingPeriod,
			"error", err.Error(),
		)
		return sdk.DecCoins{}, err
	}

	// Check for nil NFT cumulative ratios (can happen with improperly initialized historical rewards)
	if starting.NftCumulativeRewardRatio == nil {
		k.Logger(ctx).Warn("Starting period has nil NFT cumulative ratio - treating as zero",
			"validator", val.GetOperator(),
			"period", startingPeriod,
		)
		starting.NftCumulativeRewardRatio = sdk.DecCoins{}
	}
	if ending.NftCumulativeRewardRatio == nil {
		k.Logger(ctx).Warn("Ending period has nil NFT cumulative ratio - treating as zero",
			"validator", val.GetOperator(),
			"period", endingPeriod,
		)
		ending.NftCumulativeRewardRatio = sdk.DecCoins{}
	}

	difference := ending.NftCumulativeRewardRatio.Sub(starting.NftCumulativeRewardRatio)

	k.Logger(ctx).Debug("NFT reward calculation",
		"validator", val.GetOperator(),
		"starting_period", startingPeriod,
		"ending_period", endingPeriod,
		"starting_nft_ratio", starting.NftCumulativeRewardRatio.String(),
		"ending_nft_ratio", ending.NftCumulativeRewardRatio.String(),
		"difference", difference.String(),
		"nft_stake", nftStake.String(),
	)

	if difference.IsAnyNegative() {
		panic("negative NFT rewards should not be possible")
	}

	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	baseRewards := difference.MulDecTruncate(nftStake)

	// See native path; factor will be applied in multi-period variant
	_ = k.calculateProRatingFactor

	return baseRewards, nil
}
