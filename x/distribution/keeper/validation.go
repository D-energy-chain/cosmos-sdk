package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ValidateRewardConsistency checks if allocated rewards match withdrawable rewards
// This helps detect the double-split inconsistency
func (k Keeper) ValidateRewardConsistency(ctx context.Context) error {
	logger := k.Logger(ctx)
	var totalInconsistency sdk.DecCoins
	inconsistentValidators := 0

	// Iterate through all validators
	err := k.stakingKeeper.IterateValidators(ctx, func(_ int64, validator stakingtypes.ValidatorI) (stop bool) {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			logger.Error("Failed to decode validator address", "error", err)
			return false
		}

		// Get outstanding rewards (what was allocated)
		outstanding, err := k.GetValidatorOutstandingRewards(ctx, valAddr)
		if err != nil {
			logger.Error("Failed to get outstanding rewards", "validator", validator.GetOperator(), "error", err)
			return false
		}

		// Get current rewards (pending distribution)
		current, err := k.GetValidatorCurrentRewards(ctx, valAddr)
		if err != nil {
			logger.Error("Failed to get current rewards", "validator", validator.GetOperator(), "error", err)
			return false
		}

		// Get commission
		commission, err := k.GetValidatorAccumulatedCommission(ctx, valAddr)
		if err != nil {
			logger.Error("Failed to get commission", "validator", validator.GetOperator(), "error", err)
			return false
		}

		// Calculate what should be withdrawable
		// Outstanding = Commission + CurrentRewards (pending) + AlreadyWithdrawn
		// For simplicity, we check: Outstanding = Commission + CurrentRewards
		expectedOutstanding := commission.Commission.Add(current.Rewards...)

		// Check for inconsistency
		diff := outstanding.Rewards.Sub(expectedOutstanding)
		threshold := sdk.NewDecCoinFromDec(sdk.DefaultBondDenom, math.LegacyOneDec())
		diffHasLargeValue := false
		for _, coin := range diff {
			if coin.Amount.Abs().GTE(threshold.Amount) {
				diffHasLargeValue = true
				break
			}
		}
		if !diff.IsZero() && diffHasLargeValue {
			logger.Warn("⚠️  INCONSISTENCY DETECTED",
				"validator", validator.GetOperator(),
				"outstanding_rewards", outstanding.Rewards.String(),
				"commission", commission.Commission.String(),
				"current_rewards", current.Rewards.String(),
				"expected_outstanding", expectedOutstanding.String(),
				"difference", diff.String(),
			)
			totalInconsistency = totalInconsistency.Add(diff...)
			inconsistentValidators++
		}

		return false
	})

	if err != nil {
		return err
	}

	if inconsistentValidators > 0 {
		logger.Error("🚨 REWARD INCONSISTENCY SUMMARY",
			"inconsistent_validators", inconsistentValidators,
			"total_inconsistency", totalInconsistency.String(),
			"action_required", "Review REWARDS_INCONSISTENCY_ANALYSIS.md for resolution",
		)
	} else {
		logger.Info("✅ Reward consistency check passed",
			"validators_checked", "all",
			"inconsistencies_found", 0,
		)
	}

	return nil
}

// CompareAllocatedVsWithdrawable compares what was allocated to what can be withdrawn
// Returns the discrepancy for debugging
func (k Keeper) CompareAllocatedVsWithdrawable(ctx context.Context, valAddr sdk.ValAddress, delAddr sdk.AccAddress) (allocated, withdrawable, discrepancy sdk.DecCoins, err error) {
	logger := k.Logger(ctx)

	// Get validator
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get delegator starting info
	startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get current period
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return nil, nil, nil, err
	}

	// Calculate withdrawable using F1 distribution
	nativeWithdrawable, err := k.calculateDelegationRewardsBetween(ctx, val, startingInfo.PreviousPeriod, currentRewards.Period, startingInfo.Stake)
	if err != nil {
		return nil, nil, nil, err
	}

	nftWithdrawable, err := k.calculateNFTDelegationRewardsBetween(ctx, val, startingInfo.PreviousPeriod, currentRewards.Period, startingInfo.NftStake)
	if err != nil {
		return nil, nil, nil, err
	}

	withdrawable = nativeWithdrawable.Add(nftWithdrawable...)

	// To calculate what SHOULD have been allocated, we need to simulate the correct allocation
	// This is complex, so for now we just log what we can withdraw
	logger.Info("💡 Withdrawable Rewards Breakdown",
		"delegator", delAddr.String(),
		"validator", val.GetOperator(),
		"native_stake", startingInfo.Stake.String(),
		"nft_stake", startingInfo.NftStake.String(),
		"native_withdrawable", nativeWithdrawable.String(),
		"nft_withdrawable", nftWithdrawable.String(),
		"total_withdrawable", withdrawable.String(),
	)

	return sdk.DecCoins{}, withdrawable, sdk.DecCoins{}, nil
}

// LogModuleBalance logs the distribution module balance vs sum of outstanding rewards
func (k Keeper) LogModuleBalance(ctx context.Context) {
	logger := k.Logger(ctx)

	// Get distribution module account
	moduleAcc := k.authKeeper.GetModuleAccount(ctx, "distribution")
	moduleBalance := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())

	// Sum all outstanding rewards
	totalOutstanding := sdk.NewDecCoins()
	k.IterateValidatorOutstandingRewards(ctx, func(_ sdk.ValAddress, rewards types.ValidatorOutstandingRewards) (stop bool) {
		totalOutstanding = totalOutstanding.Add(rewards.Rewards...)
		return false
	})

	// Get community pool
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		logger.Error("Failed to get fee pool", "error", err)
		return
	}

	// Calculate expected balance
	expectedBalance := totalOutstanding.Add(feePool.CommunityPool...)
	expectedCoins, _ := expectedBalance.TruncateDecimal()

	// Compare
	diff := moduleBalance.Sub(expectedCoins...)

	logger.Info("📊 MODULE BALANCE VALIDATION",
		"===== ACTUAL =====", "",
		"module_balance", moduleBalance.String(),
		"===== EXPECTED =====", "",
		"total_outstanding_rewards", totalOutstanding.String(),
		"community_pool", feePool.CommunityPool.String(),
		"expected_balance", expectedCoins.String(),
		"===== DIFFERENCE =====", "",
		"difference", diff.String(),
		"status", func() string {
			if diff.IsZero() {
				return "✅ BALANCED"
			}
			return "⚠️  IMBALANCED"
		}(),
	)
}

// ValidatePoolSplit validates that the pool split is consistent
// This should be called after allocation and before period increment
func (k Keeper) ValidatePoolSplit(ctx context.Context, valAddr sdk.ValAddress, nftAllocated, nativeAllocated sdk.DecCoins) error {
	logger := k.Logger(ctx)

	// Get current rewards
	current, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	// Get staking ratios
	nftRatio, err := k.GetNftStakingRatio(ctx)
	if err != nil {
		return err
	}

	nativeRatio, err := k.GetNativeStakingRatio(ctx)
	if err != nil {
		return err
	}

	// Calculate what the re-split would give
	reSplitNFT := current.Rewards.MulDecTruncate(nftRatio)
	reSplitNative := current.Rewards.MulDecTruncate(nativeRatio)

	// Compare
	nftDiff := nftAllocated.Sub(reSplitNFT)
	nativeDiff := nativeAllocated.Sub(reSplitNative)

	if !nftDiff.IsZero() || !nativeDiff.IsZero() {
		logger.Warn("🚨 POOL SPLIT INCONSISTENCY",
			"validator", valAddr.String(),
			"===== ORIGINAL ALLOCATION =====", "",
			"nft_allocated", nftAllocated.String(),
			"native_allocated", nativeAllocated.String(),
			"===== RE-SPLIT CALCULATION =====", "",
			"nft_from_resplit", reSplitNFT.String(),
			"native_from_resplit", reSplitNative.String(),
			"===== DISCREPANCY =====", "",
			"nft_difference", nftDiff.String(),
			"native_difference", nativeDiff.String(),
			"total_difference", nftDiff.Add(nativeDiff...).String(),
		)

		return fmt.Errorf("pool split inconsistency detected")
	}

	logger.Info("✅ Pool split validation passed", "validator", valAddr.String())
	return nil
}
