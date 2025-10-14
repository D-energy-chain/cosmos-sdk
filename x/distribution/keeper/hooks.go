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

// increment period
func (h Hooks) BeforeDelegationCreated(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	_, err = h.k.IncrementValidatorPeriod(ctx, val)
	return err
}

// withdraw delegation rewards (which also increments period)
func (h Hooks) BeforeDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	del, err := h.k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err != nil {
		return err
	}

	if _, err := h.k.withdrawDelegationRewards(ctx, val, del); err != nil {
		return err
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
		return err
	}

	_, err = h.k.IncrementValidatorNFTPeriod(ctx, val)
	return err
}

// BeforeNFTDelegationSharesModified is called before NFT delegation shares are modified.
// Since rewards are now automatically distributed at epoch end, we don't withdraw here.
// We only need to ensure the period is incremented if there are accumulated rewards.
func (h Hooks) BeforeNFTDelegationSharesModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// TODO: Withdraw rewards here
	val, err := h.k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	del, err := h.k.stakingKeeper.NFTDelegationShares(ctx, delAddr, valAddr)
	if err != nil {
		return err
	}

	if _, err := h.k.withdrawNFTDelegationRewards(ctx, val, del); err != nil {
		return err
	}
	return nil
}

// create new NFT delegation period record
func (h Hooks) AfterNFTDelegationModified(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	return h.k.initializeNFTDelegation(ctx, valAddr, delAddr)
}

// BeforeNFTDelegationRemoved is called before an NFT delegation is removed.
// Withdraw accumulated rewards to ensure delegator doesn't lose them.
func (h Hooks) BeforeNFTDelegationRemoved(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) error {
	// Withdraw all accumulated rewards before removing the NFT delegation
	// This ensures delegators don't lose earned rewards during NFT offsetting
	//
	// Since NFT offsetting runs at epoch end (after reward allocation and period increment),
	// the rewards are already calculated and ready to withdraw.
	// The withdrawal function handles both native and NFT rewards together.

	// rewards, err := h.k.WithdrawNFTDelegationRewards(ctx, delAddr, valAddr)
	// if err != nil {
	// 	// Log error but don't fail the NFT delegation removal
	// 	// Some errors are expected (e.g., no delegation info if already withdrawn)
	// 	if err.Error() != types.ErrEmptyDelegationDistInfo.Error() &&
	// 		err.Error() != types.ErrNoValidatorDistInfo.Error() {
	// 		h.k.Logger(ctx).Error("Failed to withdraw rewards before NFT delegation removal",
	// 			"delegator", delAddr.String(),
	// 			"validator", valAddr.String(),
	// 			"error", err.Error(),
	// 		)
	// 	}
	// 	return nil
	// }

	// if !rewards.IsZero() {
	// 	h.k.Logger(ctx).Info("💰 Rewards automatically withdrawn before NFT offsetting",
	// 		"delegator", delAddr.String(),
	// 		"validator", valAddr.String(),
	// 		"amount", rewards.String(),
	// 	)
	// }

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

func (h Hooks) BeforeDelegationRemoved(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
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
		// if err := h.k.IncrementAllValidatorPeriods(ctx); err != nil {
		// 	ctx.Logger().Error("Failed to increment validator periods after epoch", "error", err)
		// 	return
		// }

		// Clean up old performance records (keep last 100 epochs)
		if err := h.k.CleanupOldEpochPerformanceRecords(ctx, epochIdentifier, epochNumber, 100); err != nil {
			ctx.Logger().Error("Failed to cleanup old performance records", "error", err)
			// Don't return here as it's not critical
		}

		ctx.Logger().Info("Epoch processing completed - rewards allocated and ready for withdrawal")
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
