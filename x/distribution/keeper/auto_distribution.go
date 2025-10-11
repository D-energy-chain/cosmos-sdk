package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// DistributionMetrics tracks metrics for automatic distribution
type DistributionMetrics struct {
	TotalDelegators    uint64
	SuccessfulDistribs uint64
	FailedDistribs     uint64
	TotalAmountDistrib sdk.Coins
	SkippedZeroRewards uint64
}

// DistributeRewardsToAllDelegators automatically distributes rewards to all delegators
// at epoch end. This function iterates through all delegators and transfers their
// accumulated rewards directly to their withdraw addresses.
func (k Keeper) DistributeRewardsToAllDelegators(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Initialize metrics
	metrics := DistributionMetrics{
		TotalAmountDistrib: sdk.NewCoins(),
	}

	k.Logger(ctx).Info("🎯 Starting automatic reward distribution to all delegators")

	// Iterate through all delegators
	err := k.IterateAllValidatorsAndDelegators(ctx, func(info DelegatorInfo) (stop bool) {
		metrics.TotalDelegators++

		// Distribute rewards to this specific delegator
		amount, err := k.distributeRewardsToSingleDelegator(ctx, info.DelegatorAddr, info.ValidatorAddr)
		if err != nil {
			// Log error but continue with other delegators
			k.Logger(ctx).Error("Failed to distribute rewards to delegator",
				"delegator", info.DelegatorAddr.String(),
				"validator", info.ValidatorAddr.String(),
				"error", err.Error(),
			)
			metrics.FailedDistribs++
			return false // Continue iteration
		}

		if amount.IsZero() {
			metrics.SkippedZeroRewards++
		} else {
			metrics.SuccessfulDistribs++
			metrics.TotalAmountDistrib = metrics.TotalAmountDistrib.Add(amount...)
		}

		return false // Continue iteration
	})

	if err != nil {
		k.Logger(ctx).Error("Error during delegator iteration", "error", err.Error())
		return err
	}

	// Log distribution summary
	k.Logger(ctx).Info("✅ Automatic reward distribution completed",
		"total_delegators", metrics.TotalDelegators,
		"successful_distributions", metrics.SuccessfulDistribs,
		"failed_distributions", metrics.FailedDistribs,
		"skipped_zero_rewards", metrics.SkippedZeroRewards,
		"total_amount_distributed", metrics.TotalAmountDistrib.String(),
	)

	// Emit event with distribution metrics
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute("automatic_distribution", "true"),
			sdk.NewAttribute("total_delegators", fmt.Sprintf("%d", metrics.TotalDelegators)),
			sdk.NewAttribute("successful_distributions", fmt.Sprintf("%d", metrics.SuccessfulDistribs)),
			sdk.NewAttribute("total_amount", metrics.TotalAmountDistrib.String()),
		),
	)

	return nil
}

// distributeRewardsToSingleDelegator distributes rewards to a single delegator.
// This is used both for automatic distribution and for immediate distribution
// during unbonding/removal operations.
func (k Keeper) distributeRewardsToSingleDelegator(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
) (sdk.Coins, error) {
	// Get validator
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, types.ErrNoValidatorDistInfo
	}

	// Check if delegator starting info exists
	hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	if !hasInfo {
		// No starting info means no delegation, return zero without error
		return sdk.NewCoins(), nil
	}

	// Get current rewards and period
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	// Use current period as ending period (periods already incremented at epoch end)
	endingPeriod := currentRewards.Period

	// Get delegator starting info
	startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	startingPeriod := startingInfo.PreviousPeriod
	nativeStake := startingInfo.Stake
	nftStake := startingInfo.NftStake

	// Calculate rewards
	var totalRewardsRaw sdk.DecCoins = sdk.NewDecCoins()

	// Calculate native delegation rewards if they exist
	if !nativeStake.IsZero() {
		nativeRewards, err := k.calculateDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nativeStake)
		if err != nil {
			return nil, err
		}
		totalRewardsRaw = totalRewardsRaw.Add(nativeRewards...)
	}

	// Calculate NFT delegation rewards if they exist
	if !nftStake.IsZero() {
		nftRewards, err := k.calculateNFTDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nftStake)
		if err != nil {
			return nil, err
		}
		totalRewardsRaw = totalRewardsRaw.Add(nftRewards...)
	}

	// If no rewards, return early without state changes
	if totalRewardsRaw.IsZero() {
		return sdk.NewCoins(), nil
	}

	// Get current outstanding rewards
	outstanding, err := k.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	// Apply intersection to handle rounding
	totalRewardsDecCoins := totalRewardsRaw.Intersect(outstanding)
	if !totalRewardsDecCoins.Equal(totalRewardsRaw) {
		k.Logger(ctx).Debug("Rounding adjustment during automatic distribution",
			"delegator", delAddr.String(),
			"validator", val.GetOperator(),
			"calculated", totalRewardsRaw.String(),
			"actual", totalRewardsDecCoins.String(),
		)
	}

	// Update outstanding rewards
	if !totalRewardsDecCoins.IsZero() {
		outstanding = outstanding.Sub(totalRewardsDecCoins)
		err = k.SetValidatorOutstandingRewards(ctx, valAddr, types.ValidatorOutstandingRewards{Rewards: outstanding})
		if err != nil {
			return nil, err
		}
	}

	// Convert to coins and handle remainder
	totalRewards, remainder := totalRewardsDecCoins.TruncateDecimal()
	if !remainder.IsZero() {
		// Add remainder to community pool
		feePool, err := k.FeePool.Get(ctx)
		if err != nil {
			return nil, err
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(remainder...)
		err = k.FeePool.Set(ctx, feePool)
		if err != nil {
			return nil, err
		}
	}

	// If rewards are zero after truncation, return early
	if totalRewards.IsZero() {
		return sdk.NewCoins(), nil
	}

	// Get withdraw address
	withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	// Transfer rewards to delegator's withdraw address
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, totalRewards)
	if err != nil {
		k.Logger(ctx).Error("Failed to transfer rewards during automatic distribution",
			"delegator", delAddr.String(),
			"validator", val.GetOperator(),
			"amount", totalRewards.String(),
			"error", err.Error(),
		)
		return nil, err
	}

	// Emit event for this distribution
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, totalRewards.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, valAddr.String()),
			sdk.NewAttribute(types.AttributeKeyDelegator, delAddr.String()),
		),
	)

	// Reset delegator starting info for next epoch
	// This initializes them at the current period for the next distribution cycle
	err = k.initializeDelegation(ctx, valAddr, delAddr)
	if err != nil {
		k.Logger(ctx).Error("Failed to reset delegator starting info after distribution",
			"delegator", delAddr.String(),
			"validator", val.GetOperator(),
			"error", err.Error(),
		)
		return nil, err
	}

	k.Logger(ctx).Debug("Successfully distributed rewards to delegator",
		"delegator", delAddr.String(),
		"validator", val.GetOperator(),
		"amount", totalRewards.String(),
		"withdraw_address", withdrawAddr.String(),
	)

	return totalRewards, nil
}
