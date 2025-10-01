package keeper

import (
	"context"
	"strings"
	"time"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// logPreDistributionState logs the complete validator and delegator state BEFORE any reward calculations begin
func (k Keeper) logPreDistributionState(ctx context.Context, epochNumber int64) {
	logger := k.Logger(ctx)

	logger.Info("=== PRE-DISTRIBUTION STATE SNAPSHOT ===",
		"epoch", epochNumber,
		"timestamp", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)

	// Get all validators
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		logger.Error("Failed to get validators for pre-distribution logging", "error", err)
		return
	}

	for _, validator := range validators {
		// Log native staking state
		logger.Info("Validator native staking state",
			"validator", validator.OperatorAddress,
			"tokens", validator.Tokens.String(),
			"delegator_shares", validator.DelegatorShares.String(),
			"commission_rate", validator.Commission.CommissionRates.Rate.String(),
		)

		// Log NFT staking state
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.OperatorAddress)
		if err != nil {
			logger.Error("Failed to decode validator address", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, sdk.AccAddress{}, sdk.ValAddress(valAddr))
		if err != nil {
			logger.Error("Failed to get NFT delegations", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		totalNFTShares := math.ZeroInt()
		for i, nftDel := range nftDelegations {
			logger.Info("NFT delegation pre-distribution",
				"validator", validator.OperatorAddress,
				"delegator", nftDel.DelegatorAddress,
				"contract", nftDel.NftContractAddress,
				"token_id", nftDel.TokenId,
				"shares", nftDel.Shares.String(),
				"delegation_index", i,
			)
			totalNFTShares = totalNFTShares.Add(nftDel.Shares.TruncateInt())
		}

		logger.Info("Validator NFT staking totals",
			"validator", validator.OperatorAddress,
			"total_nft_delegations", len(nftDelegations),
			"total_nft_shares", totalNFTShares.String(),
		)
	}
}

// logRewardPoolCalculations logs detailed reward pool calculations with mathematical breakdowns
func (k Keeper) logRewardPoolCalculations(ctx context.Context, validator stakingtypes.ValidatorI, totalRewards sdk.DecCoins) {
	logger := k.Logger(ctx)

	// Use DecCoins directly for calculations
	commission := totalRewards.MulDec(validator.GetCommission())
	delegatorPool := totalRewards.Sub(commission)

	// Get staking ratios
	nftStakingRatio, err := k.GetNftStakingRatio(ctx)
	if err != nil {
		logger.Error("Failed to get NFT staking ratio", "error", err)
		nftStakingRatio = math.LegacyNewDecWithPrec(75, 2) // Default 75%
	}

	nativeStakingRatio, err := k.GetNativeStakingRatio(ctx)
	if err != nil {
		logger.Error("Failed to get native staking ratio", "error", err)
		nativeStakingRatio = math.LegacyNewDecWithPrec(25, 2) // Default 25%
	}

	// Calculate native vs NFT split
	nativePool := delegatorPool.MulDecTruncate(nativeStakingRatio)
	nftPool := delegatorPool.MulDecTruncate(nftStakingRatio)

	logger.Info("Reward pool breakdown",
		"validator", validator.GetOperator(),
		"total_rewards", totalRewards.String(),
		"commission_rate", validator.GetCommission().String(),
		"commission_amount", commission.String(),
		"delegator_pool_total", delegatorPool.String(),
		"native_pool_ratio", nativeStakingRatio.String(),
		"native_pool_amount", nativePool.String(),
		"nft_pool_ratio", nftStakingRatio.String(),
		"nft_pool_amount", nftPool.String(),
	)

	// Log per-share rates
	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
	if err != nil {
		logger.Error("Failed to decode validator address for rate calculation", "error", err)
		return
	}

	nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, sdk.AccAddress{}, sdk.ValAddress(valAddr))
	if err != nil {
		logger.Error("Failed to get NFT delegations for rate calculation", "error", err)
		return
	}

	totalNFTShares := math.ZeroInt()
	for _, nftDel := range nftDelegations {
		totalNFTShares = totalNFTShares.Add(nftDel.Shares.TruncateInt())
	}

	if totalNFTShares.GT(math.ZeroInt()) && !nftPool.IsZero() {
		nftRatePerShare := nftPool[0].Amount.Quo(totalNFTShares.ToLegacyDec())
		logger.Info("NFT reward rate calculation",
			"validator", validator.GetOperator(),
			"nft_pool_amount", nftPool.String(),
			"total_nft_shares", totalNFTShares.String(),
			"rate_per_share", nftRatePerShare.String(),
		)
	}
}

// logDelegatorRewardCalculation logs each delegator's reward calculation with full mathematical breakdown
func (k Keeper) logDelegatorRewardCalculation(ctx context.Context, delegatorAddr string, validatorAddr string, shares math.LegacyDec, rewardAmount sdk.Coins, rewardType string) {
	logger := k.Logger(ctx)

	logger.Info("Delegator reward calculation",
		"delegator", delegatorAddr,
		"validator", validatorAddr,
		"reward_type", rewardType, // "native" or "nft"
		"shares", shares.String(),
		"calculated_reward", rewardAmount.String(),
		"calculation_timestamp", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)

	// For NFT delegators, also log the specific NFT details
	if rewardType == "nft" {
		delAddr, err := k.authKeeper.AddressCodec().StringToBytes(delegatorAddr)
		if err != nil {
			logger.Error("Failed to decode delegator address", "error", err)
			return
		}

		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validatorAddr)
		if err != nil {
			logger.Error("Failed to decode validator address", "error", err)
			return
		}

		nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, sdk.AccAddress(delAddr), sdk.ValAddress(valAddr))
		if err != nil {
			logger.Error("Failed to get NFT delegations for logging", "error", err)
			return
		}

		for _, nftDel := range nftDelegations {
			logger.Info("NFT delegation reward details",
				"delegator", delegatorAddr,
				"validator", validatorAddr,
				"contract", nftDel.NftContractAddress,
				"token_id", nftDel.TokenId,
				"nft_shares", nftDel.Shares.String(),
				"reward_for_this_nft", rewardAmount.String(),
			)
		}
	}
}

// logMultiValidatorDelegators tracks delegators who have stakes across multiple validators
func (k Keeper) logMultiValidatorDelegators(ctx context.Context) {
	logger := k.Logger(ctx)
	delegatorValidatorMap := make(map[string][]string) // delegator -> []validators

	// Collect all delegations
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		logger.Error("Failed to get validators for multi-validator tracking", "error", err)
		return
	}

	for _, validator := range validators {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.OperatorAddress)
		if err != nil {
			logger.Error("Failed to decode validator address for multi-validator tracking", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, sdk.AccAddress{}, sdk.ValAddress(valAddr))
		if err != nil {
			logger.Error("Failed to get NFT delegations for multi-validator tracking", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		for _, nftDel := range nftDelegations {
			delegatorValidatorMap[nftDel.DelegatorAddress] = append(
				delegatorValidatorMap[nftDel.DelegatorAddress],
				validator.OperatorAddress,
			)
		}
	}

	// Log multi-validator delegators
	for delegator, validators := range delegatorValidatorMap {
		if len(validators) > 1 {
			logger.Warn("Multi-validator delegator detected",
				"delegator", delegator,
				"validator_count", len(validators),
				"validators", strings.Join(validators, ","),
				"requires_careful_reward_tracking", true,
			)

			// Log total expected rewards across all validators
			for _, validatorAddr := range validators {
				logger.Info("Multi-validator delegator breakdown",
					"delegator", delegator,
					"validator", validatorAddr,
					"note", "reward_calculation_requires_careful_tracking",
				)
			}
		}
	}
}

// logPostDistributionValidation logs validation checks after reward distribution
func (k Keeper) logPostDistributionValidation(ctx context.Context, epochNumber int64) {
	logger := k.Logger(ctx)

	logger.Info("=== POST-DISTRIBUTION VALIDATION ===",
		"epoch", epochNumber,
	)

	// Validate total rewards distributed match expected amounts
	totalDistributed := sdk.NewCoins()
	validators, err := k.stakingKeeper.GetAllValidators(ctx)
	if err != nil {
		logger.Error("Failed to get validators for post-distribution validation", "error", err)
		return
	}

	for _, validator := range validators {
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.OperatorAddress)
		if err != nil {
			logger.Error("Failed to decode validator address for validation", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		outstandingRewards, err := k.GetValidatorOutstandingRewards(ctx, sdk.ValAddress(valAddr))
		if err != nil {
			logger.Error("Failed to get outstanding rewards", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		outstandingCoins, _ := outstandingRewards.Rewards.TruncateDecimal()
		totalDistributed = totalDistributed.Add(outstandingCoins...)

		logger.Info("Validator post-distribution state",
			"validator", validator.OperatorAddress,
			"outstanding_rewards", outstandingRewards.Rewards.String(),
		)

		// Check if NFT delegations were properly updated after offset
		nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, sdk.AccAddress{}, sdk.ValAddress(valAddr))
		if err != nil {
			logger.Error("Failed to get NFT delegations for validation", "validator", validator.OperatorAddress, "error", err)
			continue
		}

		logger.Info("Validator post-offset NFT state",
			"validator", validator.OperatorAddress,
			"nft_delegations_count", len(nftDelegations),
		)

		for _, nftDel := range nftDelegations {
			logger.Info("NFT delegation post-offset",
				"validator", validator.OperatorAddress,
				"delegator", nftDel.DelegatorAddress,
				"contract", nftDel.NftContractAddress,
				"token_id", nftDel.TokenId,
				"shares_after_offset", nftDel.Shares.String(),
			)
		}
	}

	logger.Info("Total rewards validation",
		"epoch", epochNumber,
		"total_distributed", totalDistributed.String(),
	)
}

// validateRewardCalculations validates reward calculations with detailed error logging
func (k Keeper) validateRewardCalculations(ctx context.Context, delegator string, expectedReward, actualReward sdk.Coins) {
	logger := k.Logger(ctx)

	if !expectedReward.Equal(actualReward) {
		variance := actualReward.Sub(expectedReward...)
		var percentageVariance math.LegacyDec

		if !expectedReward.IsZero() && !expectedReward[0].Amount.IsZero() {
			percentageVariance = variance[0].Amount.ToLegacyDec().Quo(expectedReward[0].Amount.ToLegacyDec()).MulInt64(100)
		}

		logLevel := "Info"
		if percentageVariance.Abs().GT(math.LegacyNewDecFromInt(math.NewInt(10))) { // >10% variance
			logLevel = "Error"
		} else if percentageVariance.Abs().GT(math.LegacyNewDecFromInt(math.NewInt(5))) { // >5% variance
			logLevel = "Warn"
		}

		switch logLevel {
		case "Error":
			logger.Error("CRITICAL: Reward calculation variance detected",
				"delegator", delegator,
				"expected_reward", expectedReward.String(),
				"actual_reward", actualReward.String(),
				"variance", variance.String(),
				"percentage_variance", percentageVariance.String()+"%",
				"requires_investigation", true,
			)
		case "Warn":
			logger.Warn("Moderate reward calculation variance",
				"delegator", delegator,
				"expected_reward", expectedReward.String(),
				"actual_reward", actualReward.String(),
				"variance", variance.String(),
				"percentage_variance", percentageVariance.String()+"%",
			)
		default:
			logger.Info("Minor reward calculation variance",
				"delegator", delegator,
				"variance", variance.String(),
				"percentage_variance", percentageVariance.String()+"%",
			)
		}
	}
}

// logRewardDistributionSummary logs a comprehensive summary of the reward distribution process
func (k Keeper) logRewardDistributionSummary(ctx context.Context, epochNumber int64, totalFeesCollected sdk.Coins, validatorsProcessed int, performanceBased bool) {
	logger := k.Logger(ctx)

	logger.Info("=== REWARD DISTRIBUTION SUMMARY ===",
		"epoch", epochNumber,
		"total_fees_collected", totalFeesCollected.String(),
		"validators_processed", validatorsProcessed,
		"performance_based", performanceBased,
		"distribution_completed_at", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)
}

// logValidatorRewardAllocation logs detailed reward allocation for each validator
func (k Keeper) logValidatorRewardAllocation(ctx context.Context, validator stakingtypes.ValidatorI, totalRewards sdk.DecCoins, nftRewards, nativeRewards sdk.DecCoins, commission sdk.DecCoins) {
	logger := k.Logger(ctx)

	logger.Info("Validator reward allocation",
		"validator", validator.GetOperator(),
		"total_rewards", totalRewards.String(),
		"nft_rewards", nftRewards.String(),
		"native_rewards", nativeRewards.String(),
		"commission", commission.String(),
		"commission_rate", validator.GetCommission().String(),
	)
}

// logDelegationRewardWithdrawal logs when rewards are withdrawn by delegators
func (k Keeper) logDelegationRewardWithdrawal(ctx context.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress, rewards sdk.Coins, rewardType string) {
	logger := k.Logger(ctx)

	logger.Info("Delegation reward withdrawal",
		"delegator", delegatorAddr.String(),
		"validator", validatorAddr.String(),
		"reward_type", rewardType,
		"withdrawn_amount", rewards.String(),
		"withdrawal_timestamp", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)
}

// logPeriodIncrement logs when validator periods are incremented
func (k Keeper) logPeriodIncrement(ctx context.Context, validatorAddr sdk.ValAddress, oldPeriod, newPeriod uint64, reason string) {
	logger := k.Logger(ctx)

	logger.Info("Validator period increment",
		"validator", validatorAddr.String(),
		"old_period", oldPeriod,
		"new_period", newPeriod,
		"reason", reason,
		"increment_timestamp", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)
}

// logSlashEvent logs when validators are slashed
func (k Keeper) logSlashEvent(ctx context.Context, validatorAddr sdk.ValAddress, fraction math.LegacyDec, period uint64, height uint64) {
	logger := k.Logger(ctx)

	logger.Warn("Validator slash event",
		"validator", validatorAddr.String(),
		"slash_fraction", fraction.String(),
		"period", period,
		"height", height,
		"slash_timestamp", sdk.UnwrapSDKContext(ctx).BlockTime().Format(time.RFC3339),
	)
}
