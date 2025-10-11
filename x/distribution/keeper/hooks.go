package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Wrapper struct
type Hooks struct {
	k Keeper
}

var _ stakingtypes.StakingHooks = Hooks{}

var _ types.EpochHooks = Hooks{}

var _ types.NFTStakingHooks = Hooks{}

// Create new distribution hooks
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// initialize validator distribution record
func (h Hooks) AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}
	return h.k.initializeValidator(ctx, val)
}

// AfterValidatorRemoved performs clean up after a validator is removed
func (h Hooks) AfterValidatorRemoved(ctx context.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) error {
	// fetch outstanding
	outstanding, err := h.k.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
	if err != nil {
		return err
	}

	// force-withdraw commission
	valCommission, err := h.k.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return err
	}

	commission := valCommission.Commission

	if !commission.IsZero() {
		// subtract from outstanding
		outstanding = outstanding.Sub(commission)

		// split into integral & remainder
		coins, remainder := commission.TruncateDecimal()

		// remainder to community pool
		feePool, err := h.k.FeePool.Get(ctx)
		if err != nil {
			return err
		}

		feePool.CommunityPool = feePool.CommunityPool.Add(remainder...)
		err = h.k.FeePool.Set(ctx, feePool)
		if err != nil {
			return err
		}

		// add to validator account
		if !coins.IsZero() {
			accAddr := sdk.AccAddress(valAddr)
			withdrawAddr, err := h.k.GetDelegatorWithdrawAddr(ctx, accAddr)
			if err != nil {
				return err
			}

			if err := h.k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, coins); err != nil {
				return err
			}
		}
	}

	// Add outstanding to community pool
	// The validator is removed only after it has no more delegations.
	// This operation sends only the remaining dust to the community pool.
	feePool, err := h.k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(outstanding...)
	err = h.k.FeePool.Set(ctx, feePool)
	if err != nil {
		return err
	}

	// delete outstanding
	err = h.k.DeleteValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	// remove commission record
	err = h.k.DeleteValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return err
	}

	// clear slashes
	h.k.DeleteValidatorSlashEvents(ctx, valAddr)

	// clear historical rewards
	h.k.DeleteValidatorHistoricalRewards(ctx, valAddr)

	// clear current rewards
	err = h.k.DeleteValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	return nil
}

// increment period - now only on delegation creation for immediate effect
// Note: Main period increments happen at epoch end for epoch-based rewards
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	// Only increment period if there are accumulated rewards that need to be locked in
	// This handles the case where someone delegates mid-epoch
	currentRewards, err := h.k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	// If there are accumulated rewards, increment period to lock them in
	// before the new delegation affects the reward calculation
	if !currentRewards.Rewards.IsZero() {
		_, err = h.k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			return err
		}
	}

	return nil
}

// BeforeDelegationSharesModified is called before delegation shares are modified.
// Since rewards are now automatically distributed at epoch end, we don't withdraw here.
// We only need to ensure the period is incremented if there are accumulated rewards,
// so that the delegator's starting info is updated correctly for pro-rated rewards.
func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	// Increment period if there are accumulated rewards
	// This locks in rewards before the share change affects calculations
	currentRewards, err := h.k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	if !currentRewards.Rewards.IsZero() {
		_, err = h.k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			return err
		}
	}

	return nil
}

// create new delegation period record
func (h Hooks) AfterDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return h.k.initializeDelegation(ctx, valAddr, delAddr)
}

// NFT staking hooks - these are called by the NFT staking module

// increment period - now only on NFT delegation creation for immediate effect
// Note: Main period increments happen at epoch end for epoch-based rewards
func (h Hooks) BeforeNFTDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		h.k.Logger(ctx).Error("Failed to get validator for NFT delegation", "error", err)
		return err
	}

	// Only increment period if there are accumulated rewards that need to be locked in
	// This handles the case where someone delegates mid-epoch
	currentRewards, err := h.k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		h.k.Logger(ctx).Error("Failed to get current rewards for NFT delegation", "error", err)
		return err
	}

	// If there are accumulated rewards, increment period to lock them in
	// before the new NFT delegation affects the reward calculation
	if !currentRewards.Rewards.IsZero() {
		_, err = h.k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			h.k.Logger(ctx).Error("Failed to increment period for NFT delegation", "error", err)
			return err
		}
	}

	return nil
}

// BeforeNFTDelegationSharesModified is called before NFT delegation shares are modified.
// Since rewards are now automatically distributed at epoch end, we don't withdraw here.
// We only need to ensure the period is incremented if there are accumulated rewards.
func (h Hooks) BeforeNFTDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		h.k.Logger(ctx).Error("Failed to get validator for NFT delegation shares modification", "error", err)
		return err
	}

	// Increment period if there are accumulated rewards
	// This locks in rewards before the share change affects calculations
	currentRewards, err := h.k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		h.k.Logger(ctx).Error("Failed to get current rewards for NFT delegation", "error", err)
		return err
	}

	if !currentRewards.Rewards.IsZero() {
		_, err = h.k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			h.k.Logger(ctx).Error("Failed to increment period for NFT delegation", "error", err)
			return err
		}
	}

	return nil
}

// create new NFT delegation period record
func (h Hooks) AfterNFTDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return h.k.initializeDelegation(ctx, valAddr, delAddr)
}

// BeforeNFTDelegationRemoved is called before an NFT delegation is removed.
// Immediately distribute any accumulated rewards to ensure delegator doesn't lose them.
func (h Hooks) BeforeNFTDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// Check if delegator starting info exists
	hasInfo, err := h.k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return err
	}

	// If starting info exists, distribute any accumulated rewards
	if hasInfo {
		// Distribute NFT delegation rewards immediately before removal (zero threshold = no minimum)
		if _, err := h.k.distributeRewardsToSingleDelegator(ctx, delAddr, valAddr, sdkmath.LegacyZeroDec()); err != nil {
			// Log error but don't fail the delegation removal
			h.k.Logger(ctx).Error("Failed to distribute NFT delegation rewards before removal", "error", err)
		}
	}

	return nil
}

// record the slash event
func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, fraction sdkmath.LegacyDec) error {
	return h.k.updateValidatorSlashFraction(ctx, valAddr, fraction)
}

func (h Hooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBonded(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) AfterValidatorBeginUnbonding(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h Hooks) BeforeDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// Immediately distribute any accumulated rewards before removing the delegation
	// This ensures the delegator doesn't lose their earned rewards
	// Uses zero threshold to distribute all rewards immediately regardless of amount

	// Check if delegation exists before attempting distribution
	del, err := h.k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err != nil {
		// If delegation doesn't exist, nothing to distribute
		return nil
	}

	if del == nil {
		// No delegation found, nothing to distribute
		return nil
	}

	// Distribute delegation rewards immediately before removal (zero threshold = no minimum)
	if _, err := h.k.distributeRewardsToSingleDelegator(ctx, delAddr, valAddr, sdkmath.LegacyZeroDec()); err != nil {
		// Log error but don't fail the delegation removal
		h.k.Logger(ctx).Error("Failed to distribute delegation rewards before removal", "error", err)
	}

	return nil
}

func (h Hooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error {
	return nil
}

// BeforeEpochStart: noop, We don't need to do anything here
func (h Hooks) BeforeEpochStart(_ sdk.Context, _ string, _ int64) {
}

// AfterEpochEnd mints and allocates coins at the end of each epoch end
func (h Hooks) AfterEpochEnd(ctx sdk.Context, epochIdentifier string, epochNumber int64) {
	// get distribution parameters to check if performance-based distribution is enabled
	params, err := h.k.Params.Get(ctx)
	if err != nil {
		ctx.Logger().Error("Failed to get distribution params", "error", err)
		return
	}

	ctx.Logger().Info("Epoch ended - starting reward distribution",
		"epoch_identifier", epochIdentifier,
		"epoch_number", epochNumber,
		"performance_based", params.EnablePerformanceBasedDistribution)

	if params.EnablePerformanceBasedDistribution {
		// Use performance-based allocation
		if err := h.k.AllocateTokensWithPerformance(ctx, epochIdentifier, epochNumber); err != nil {
			ctx.Logger().Error("Failed to allocate tokens using performance-based distribution", "error", err)
			return
		}

		// CRITICAL: Increment periods for all validators to lock in this epoch's rewards
		// This converts accumulated rewards to cumulative ratios for proper F1 distribution
		if err := h.k.IncrementAllValidatorPeriods(ctx); err != nil {
			ctx.Logger().Error("Failed to increment validator periods after epoch", "error", err)
			return
		}

		// Automatically distribute rewards to all delegators
		// This transfers accumulated rewards directly to delegator withdraw addresses
		if err := h.k.DistributeRewardsToAllDelegators(ctx); err != nil {
			ctx.Logger().Error("Failed to automatically distribute rewards to delegators", "error", err)
			// Don't return here - epoch end should complete even if distribution fails
			// Rewards remain in outstanding and can be distributed next epoch
		}

		// Clean up old performance records (keep last 100 epochs)
		if err := h.k.CleanupOldEpochPerformanceRecords(ctx, epochIdentifier, epochNumber, 100); err != nil {
			ctx.Logger().Error("Failed to cleanup old performance records", "error", err)
			// Don't return here as it's not critical
		}

		ctx.Logger().Info("Performance-based reward distribution completed successfully")
	} else {
		// Use legacy single-block voting allocation
		var previousTotalPower int64
		for _, voteInfo := range ctx.VoteInfos() {
			previousTotalPower += voteInfo.Validator.Power
		}

		// record the proposer for when we payout on the next block
		consAddr := sdk.ConsAddress(ctx.BlockHeader().ProposerAddress)
		if err := h.k.SetPreviousProposerConsAddr(ctx, consAddr); err != nil {
			ctx.Logger().Error("Failed to set proposer from block header", "error", err)
			return
		}

		// At epoch end, distribute all accumulated rewards using legacy method
		if err := h.k.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos()); err != nil {
			ctx.Logger().Error("Failed to allocate tokens during epoch end", "error", err)
			return
		}

		ctx.Logger().Info("Legacy epoch-based reward distribution completed successfully")
	}
}
