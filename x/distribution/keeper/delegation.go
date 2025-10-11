package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// resetDelegatorStartingInfoToPeriod resets the delegator's starting info to a specific period
// after rewards have been distributed. This is used after automatic distribution to set the
// starting period to the ending period that was just used for calculation.
func (k Keeper) resetDelegatorStartingInfoToPeriod(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress, period uint64) error {
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
func (k Keeper) initializeDelegation(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress) error {
	// Check if starting info already exists - if so, don't initialize again
	hasInfo, err := k.HasDelegatorStartingInfo(ctx, val, del)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// period has already been incremented - we want to store the period ended by this delegation action
	valCurrentRewards, err := k.GetValidatorCurrentRewards(ctx, val)
	if err != nil {
		return err
	}
	previousPeriod := valCurrentRewards.Period - 1

	if hasInfo {
		// Starting info already exists - update the NFT stake if needed

		// Get existing starting info
		existingInfo, err := k.GetDelegatorStartingInfo(ctx, val, del)
		if err != nil {
			return err
		}

		// Check if we need to update NFT stake
		var currentNftStake math.LegacyDec = math.LegacyZeroDec()
		nftShares, err := k.stakingKeeper.GetNFTDelegatorShares(ctx, del, val)
		if err == nil {
			currentNftStake = nftShares
		}

		// If NFT stake changed, update the starting info
		if !currentNftStake.Equal(existingInfo.NftStake) {
			existingInfo.NftStake = currentNftStake
			return k.SetDelegatorStartingInfo(ctx, val, del, existingInfo)
		}
		return nil
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
	return k.SetDelegatorStartingInfo(ctx, val, del, startingInfo)
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx context.Context, val stakingtypes.ValidatorI,
	startingPeriod, endingPeriod uint64, stake math.LegacyDec,
) (sdk.DecCoins, error) {
	// sanity check
	if startingPeriod > endingPeriod {
		panic("startingPeriod cannot be greater than endingPeriod")
	}

	// sanity check
	if stake.IsNegative() {
		panic("stake should not be negative")
	}

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		panic(err)
	}

	// return staking * (ending - starting)
	starting, err := k.GetValidatorHistoricalRewards(ctx, valBz, startingPeriod)
	if err != nil {
		k.Logger(ctx).Error("Missing starting period historical rewards",
			"validator", val.GetOperator(),
			"period", startingPeriod,
			"error", err.Error(),
		)
		return sdk.DecCoins{}, err
	}

	ending, err := k.GetValidatorHistoricalRewards(ctx, valBz, endingPeriod)
	if err != nil {
		k.Logger(ctx).Error("Missing ending period historical rewards",
			"validator", val.GetOperator(),
			"period", endingPeriod,
			"error", err.Error(),
		)
		return sdk.DecCoins{}, err
	}

	difference := ending.CumulativeRewardRatio.Sub(starting.CumulativeRewardRatio)

	k.Logger(ctx).Debug("Native reward calculation",
		"validator", val.GetOperator(),
		"starting_period", startingPeriod,
		"ending_period", endingPeriod,
		"starting_ratio", starting.CumulativeRewardRatio.String(),
		"ending_ratio", ending.CumulativeRewardRatio.String(),
		"difference", difference.String(),
		"stake", stake.String(),
	)

	if difference.IsAnyNegative() {
		panic("negative rewards should not be possible")
	}
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	baseRewards := difference.MulDecTruncate(stake)

	// Pro-rate using heights captured at historical periods
	// The period N represents rewards accumulated up to period N, and its Height
	// corresponds to the block height when period N was created.
	proRateFactor := k.calculateProRatingFactor(
		ctx,
		0, // placeholder; real delegation height applied in multi-period path
		starting.Height,
		ending.Height,
	)
	_ = proRateFactor // For now, base function remains unchanged; multi-period variant will use factor

	return baseRewards, nil
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

	// return nft_staking * (ending - starting) using NFT cumulative reward ratio
	starting, err := k.GetValidatorHistoricalRewards(ctx, valBz, startingPeriod)
	if err != nil {
		k.Logger(ctx).Error("Missing starting period historical rewards (NFT)",
			"validator", val.GetOperator(),
			"period", startingPeriod,
			"error", err.Error(),
		)
		return sdk.DecCoins{}, err
	}

	ending, err := k.GetValidatorHistoricalRewards(ctx, valBz, endingPeriod)
	if err != nil {
		k.Logger(ctx).Error("Missing ending period historical rewards (NFT)",
			"validator", val.GetOperator(),
			"period", endingPeriod,
			"error", err.Error(),
		)
		return sdk.DecCoins{}, err
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

// calculateDelegationRewardsBetweenWithProRating calculates rewards across periods
// and applies height-based pro-rating per period using the delegator's creation height.
func (k Keeper) calculateDelegationRewardsBetweenWithProRating(
	ctx context.Context,
	val stakingtypes.ValidatorI,
	startingPeriod uint64,
	endingPeriod uint64,
	stake math.LegacyDec,
	delegationHeight uint64,
) (sdk.DecCoins, error) {
	if startingPeriod > endingPeriod {
		panic("startingPeriod cannot be greater than endingPeriod")
	}

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return sdk.DecCoins{}, err
	}

	total := sdk.NewDecCoins()

	for period := startingPeriod; period < endingPeriod; period++ {
		startRec, err := k.GetValidatorHistoricalRewards(ctx, valBz, period)
		if err != nil {
			return sdk.DecCoins{}, err
		}
		endRec, err := k.GetValidatorHistoricalRewards(ctx, valBz, period+1)
		if err != nil {
			return sdk.DecCoins{}, err
		}

		diff := endRec.CumulativeRewardRatio.Sub(startRec.CumulativeRewardRatio)
		base := diff.MulDecTruncate(stake)

		factor := k.calculateProRatingFactor(ctx, delegationHeight, startRec.Height, endRec.Height)
		prorated := base.MulDecTruncate(factor)
		// Optional: log per-period details for debugging/auditing
		k.Logger(ctx).Debug("📊 Pro-rating calculation",
			"delegation_height", delegationHeight,
			"period", period,
			"period_start_height", startRec.Height,
			"period_end_height", endRec.Height,
			"pro_rate_factor", factor.String(),
			"base_rewards", base.String(),
			"pro_rated_rewards", prorated.String(),
		)
		total = total.Add(prorated...)
	}

	return total, nil
}

// calculateNFTDelegationRewardsBetweenWithProRating applies the same logic for NFT stake.
func (k Keeper) calculateNFTDelegationRewardsBetweenWithProRating(
	ctx context.Context,
	val stakingtypes.ValidatorI,
	startingPeriod uint64,
	endingPeriod uint64,
	nftStake math.LegacyDec,
	delegationHeight uint64,
) (sdk.DecCoins, error) {
	if startingPeriod > endingPeriod {
		panic("startingPeriod cannot be greater than endingPeriod")
	}

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return sdk.DecCoins{}, err
	}

	total := sdk.NewDecCoins()

	for period := startingPeriod; period < endingPeriod; period++ {
		startRec, err := k.GetValidatorHistoricalRewards(ctx, valBz, period)
		if err != nil {
			return sdk.DecCoins{}, err
		}
		endRec, err := k.GetValidatorHistoricalRewards(ctx, valBz, period+1)
		if err != nil {
			return sdk.DecCoins{}, err
		}

		diff := endRec.NftCumulativeRewardRatio.Sub(startRec.NftCumulativeRewardRatio)
		base := diff.MulDecTruncate(nftStake)

		factor := k.calculateProRatingFactor(ctx, delegationHeight, startRec.Height, endRec.Height)
		prorated := base.MulDecTruncate(factor)
		// Optional: log per-period details for debugging/auditing (NFT)
		k.Logger(ctx).Debug("📊 Pro-rating calculation (NFT)",
			"delegation_height", delegationHeight,
			"period", period,
			"period_start_height", startRec.Height,
			"period_end_height", endRec.Height,
			"pro_rate_factor", factor.String(),
			"base_rewards", base.String(),
			"pro_rated_rewards", prorated.String(),
		)
		total = total.Add(prorated...)
	}

	return total, nil
}

// calculate the total rewards accrued by a delegation
func (k Keeper) CalculateDelegationRewards(ctx context.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI, endingPeriod uint64) (rewards sdk.DecCoins, err error) {
	addrCodec := k.authKeeper.AddressCodec()
	delAddr, err := addrCodec.StringToBytes(del.GetDelegatorAddr())
	if err != nil {
		return sdk.DecCoins{}, err
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
	if err != nil {
		return sdk.DecCoins{}, err
	}

	// fetch starting info for delegation
	startingInfo, err := k.GetDelegatorStartingInfo(ctx, sdk.ValAddress(valAddr), sdk.AccAddress(delAddr))
	if err != nil {
		return
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if startingInfo.Height == uint64(sdkCtx.BlockHeight()) {
		// started this height, no rewards yet
		return
	}

	startingPeriod := startingInfo.PreviousPeriod
	stake := startingInfo.Stake
	nftStake := startingInfo.NftStake

	// Iterate through slashes and withdraw with calculated staking for
	// distribution periods. These period offsets are dependent on *when* slashes
	// happen - namely, in BeginBlock, after rewards are allocated...
	// Slashes which happened in the first block would have been before this
	// delegation existed, UNLESS they were slashes of a redelegation to this
	// validator which was itself slashed (from a fault committed by the
	// redelegation source validator) earlier in the same BeginBlock.
	startingHeight := startingInfo.Height
	// Slashes this block happened after reward allocation, but we have to account
	// for them for the stake sanity check below.
	endingHeight := uint64(sdkCtx.BlockHeight())
	if endingHeight > startingHeight {
		k.IterateValidatorSlashEventsBetween(ctx, valAddr, startingHeight, endingHeight,
			func(height uint64, event types.ValidatorSlashEvent) (stop bool) {
				endingPeriod := event.ValidatorPeriod
				if endingPeriod > startingPeriod {
					// Calculate native delegation rewards with height-based pro-rating
					delRewards, err := k.calculateDelegationRewardsBetweenWithProRating(ctx, val, startingPeriod, endingPeriod, stake, startingInfo.Height)
					if err != nil {
						panic(err)
					}
					rewards = rewards.Add(delRewards...)

					// Calculate NFT delegation rewards with height-based pro-rating
					nftDelRewards, err := k.calculateNFTDelegationRewardsBetweenWithProRating(ctx, val, startingPeriod, endingPeriod, nftStake, startingInfo.Height)
					if err != nil {
						panic(err)
					}
					rewards = rewards.Add(nftDelRewards...)

					// Note: It is necessary to truncate so we don't allow withdrawing
					// more rewards than owed.
					stake = stake.MulTruncate(math.LegacyOneDec().Sub(event.Fraction))
					nftStake = nftStake.MulTruncate(math.LegacyOneDec().Sub(event.Fraction))
					startingPeriod = endingPeriod
				}
				return false
			},
		)
	}

	// A total stake sanity check; Recalculated final stake should be less than or
	// equal to current stake here. We cannot use Equals because stake is truncated
	// when multiplied by slash fractions (see above). We could only use equals if
	// we had arbitrary-precision rationals.
	currentStake := val.TokensFromShares(del.GetShares())

	if stake.GT(currentStake) {
		// AccountI for rounding inconsistencies between:
		//
		//     currentStake: calculated as in staking with a single computation
		//     stake:        calculated as an accumulation of stake
		//                   calculations across validator's distribution periods
		//
		// These inconsistencies are due to differing order of operations which
		// will inevitably have different accumulated rounding and may lead to
		// the smallest decimal place being one greater in stake than
		// currentStake. When we calculated slashing by period, even if we
		// round down for each slash fraction, it's possible due to how much is
		// being rounded that we slash less when slashing by period instead of
		// for when we slash without periods. In other words, the single slash,
		// and the slashing by period could both be rounding down but the
		// slashing by period is simply rounding down less, thus making stake >
		// currentStake
		//
		// A small amount of this error is tolerated and corrected for,
		// however any greater amount should be considered a breach in expected
		// behavior.
		marginOfErr := math.LegacySmallestDec().MulInt64(3)
		if stake.LTE(currentStake.Add(marginOfErr)) {
			stake = currentStake
		} else {
			panic(fmt.Sprintf("calculated final stake for delegator %s greater than current stake"+
				"\n\tfinal stake:\t%s"+
				"\n\tcurrent stake:\t%s",
				del.GetDelegatorAddr(), stake, currentStake))
		}
	}

	// calculate rewards for final period - both native and NFT with pro-rating
	delRewards, err := k.calculateDelegationRewardsBetweenWithProRating(ctx, val, startingPeriod, endingPeriod, stake, startingInfo.Height)
	if err != nil {
		return sdk.DecCoins{}, err
	}
	rewards = rewards.Add(delRewards...)

	// calculate NFT rewards for final period with pro-rating
	nftDelRewards, err := k.calculateNFTDelegationRewardsBetweenWithProRating(ctx, val, startingPeriod, endingPeriod, nftStake, startingInfo.Height)
	if err != nil {
		return sdk.DecCoins{}, err
	}
	rewards = rewards.Add(nftDelRewards...)

	// Log individual delegator reward calculations
	if !delRewards.IsZero() {
		delRewardsCoins, _ := delRewards.TruncateDecimal()
		k.logDelegatorRewardCalculation(ctx, del.GetDelegatorAddr(), val.GetOperator(), stake, delRewardsCoins, "native")
	}
	if !nftDelRewards.IsZero() {
		nftDelRewardsCoins, _ := nftDelRewards.TruncateDecimal()
		k.logDelegatorRewardCalculation(ctx, del.GetDelegatorAddr(), val.GetOperator(), nftStake, nftDelRewardsCoins, "nft")
	}

	return rewards, nil
}

// DEPRECATED: This function has been replaced by the fixed version in keeper.go
// Keeping it commented out for reference but it should not be used
/*
func (k Keeper) withdrawDelegationRewards(ctx context.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI) (sdk.Coins, error) {
	addrCodec := k.authKeeper.AddressCodec()
	delAddr, err := addrCodec.StringToBytes(del.GetDelegatorAddr())
	if err != nil {
		return nil, err
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
	if err != nil {
		return nil, err
	}

	// check existence of delegator starting info
	hasInfo, err := k.HasDelegatorStartingInfo(ctx, sdk.ValAddress(valAddr), sdk.AccAddress(delAddr))
	if err != nil {
		return nil, err
	}

	if !hasInfo {
		return nil, types.ErrEmptyDelegationDistInfo
	}

	// end current period and calculate rewards
	endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return nil, err
	}

	rewardsRaw, err := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	if err != nil {
		return nil, err
	}

	outstanding, err := k.GetValidatorOutstandingRewardsCoins(ctx, sdk.ValAddress(valAddr))
	if err != nil {
		return nil, err
	}

	// defensive edge case may happen on the very final digits
	// of the decCoins due to operation order of the distribution mechanism.
	rewards := rewardsRaw.Intersect(outstanding)
	if !rewards.Equal(rewardsRaw) {
		logger := k.Logger(ctx)
		logger.Info(
			"rounding error withdrawing rewards from validator",
			"delegator", del.GetDelegatorAddr(),
			"validator", val.GetOperator(),
			"got", rewards.String(),
			"expected", rewardsRaw.String(),
		)
	}

	// truncate reward dec coins, return remainder to community pool
	finalRewards, remainder := rewards.TruncateDecimal()

	// add coins to user account
	if !finalRewards.IsZero() {
		withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, finalRewards)
		if err != nil {
			return nil, err
		}

		// Log delegation reward withdrawal
		k.logDelegationRewardWithdrawal(ctx, sdk.AccAddress(delAddr), sdk.ValAddress(valAddr), finalRewards, "native")
	}

	// update the outstanding rewards and the community pool only if the
	// transaction was successful
	err = k.SetValidatorOutstandingRewards(ctx, sdk.ValAddress(valAddr), types.ValidatorOutstandingRewards{Rewards: outstanding.Sub(rewards)})
	if err != nil {
		return nil, err
	}

	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return nil, err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(remainder...)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		return nil, err
	}

	// decrement reference count of starting period
	startingInfo, err := k.GetDelegatorStartingInfo(ctx, sdk.ValAddress(valAddr), sdk.AccAddress(delAddr))
	if err != nil {
		return nil, err
	}

	startingPeriod := startingInfo.PreviousPeriod
	err = k.decrementReferenceCount(ctx, sdk.ValAddress(valAddr), startingPeriod)
	if err != nil {
		return nil, err
	}

	// remove delegator starting info
	err = k.DeleteDelegatorStartingInfo(ctx, sdk.ValAddress(valAddr), sdk.AccAddress(delAddr))
	if err != nil {
		return nil, err
	}

	if finalRewards.IsZero() {
		baseDenom, _ := sdk.GetBaseDenom()
		if baseDenom == "" {
			baseDenom = sdk.DefaultBondDenom
		}

		// Note, we do not call the NewCoins constructor as we do not want the zero
		// coin removed.
		finalRewards = sdk.Coins{sdk.NewCoin(baseDenom, math.ZeroInt())}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, finalRewards.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator()),
			sdk.NewAttribute(types.AttributeKeyDelegator, del.GetDelegatorAddr()),
		),
	)

	return finalRewards, nil
}
*/
