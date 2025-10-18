package keeper

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AllocateTokens performs reward and fee distribution to all validators based
// on the F1 fee distribution specification.
func (k Keeper) AllocateTokens(ctx context.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) error {
	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	if feesCollectedInt.IsZero() {
		return nil
	}

	// Log pre-distribution state
	k.logPreDistributionState(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		k.Logger(ctx).Error("Failed to transfer fees to distribution module", "error", err)
		return err
	}

	k.Logger(ctx).Info("Allocated fees to validators", "amount", feesCollectedInt)

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// All fees go directly to validators (no community tax)
	remaining := feesCollected
	feeMultiplier := feesCollected // 100% goes to validators

	// allocate tokens proportionally to voting power. Validators that did not
	// sign the last block (i.e. votes with BlockIDFlagCommit != commit) are
	// skipped, which effectively results in a micro-slash because their share of
	// the rewards is redirected to the community pool.
	//
	// Ref: https://github.com/cosmos/cosmos-sdk/pull/3099#discussion_r246276376
	validatorsProcessed := 0
	for _, vote := range bondedVotes {
		// Skip validators that missed their vote for the previous block.
		if vote.BlockIdFlag != cmtproto.BlockIDFlagCommit {
			continue
		}

		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		if err != nil {
			k.Logger(ctx).Error("Failed to get validator by consensus address", "error", err)
			return err
		}

		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		reward := feeMultiplier.MulDecTruncate(powerFraction)

		// Log reward pool calculations for this validator
		k.logRewardPoolCalculations(ctx, validator, reward)

		err = k.AllocateTokensToValidator(ctx, validator, reward)
		if err != nil {
			k.Logger(ctx).Error("Failed to allocate tokens to validator", "validator", validator.GetOperator(), "error", err)
			return err
		}

		remaining = remaining.Sub(reward)
		validatorsProcessed++
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		k.Logger(ctx).Error("Failed to set fee pool", "error", err)
		return err
	}

	// Log distribution summary
	k.logRewardDistributionSummary(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight(), feesCollectedInt, validatorsProcessed, false)

	// Log multi-validator delegators
	k.logMultiValidatorDelegators(ctx)

	// Log post-distribution validation
	k.logPostDistributionValidation(ctx, sdk.UnwrapSDKContext(ctx).BlockHeight())

	return nil
}

// AllocateTokensWithPerformance performs reward and fee distribution based on epoch performance
// rather than single-block voting status. This provides fairer distribution for epoch-based systems.
func (k Keeper) AllocateTokensWithPerformance(ctx context.Context, epochIdentifier string, epochNumber int64) error {
	// fetch and clear the collected fees for distribution
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	if feesCollectedInt.IsZero() {
		return nil
	}

	// Log pre-distribution state
	k.logPreDistributionState(ctx, epochNumber)

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		k.Logger(ctx).Error("Failed to transfer fees to distribution module", "error", err)
		return err
	}

	// get distribution parameters
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// get all validator performance records for this epoch
	performances, err := k.GetAllValidatorEpochPerformanceForEpoch(ctx, epochIdentifier, epochNumber)
	if err != nil {
		k.Logger(ctx).Error("Failed to get validator epoch performances", "error", err)
		return err
	}

	if len(performances) == 0 {
		feePool, err := k.FeePool.Get(ctx)
		if err != nil {
			return err
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// calculate total weighted power from eligible validators
	// IMPORTANT: We use a linear performance adjustment to avoid quadratic penalty effects.
	// The formula is: weightedPower = power * (baseWeight + performanceWeight * commitRatio)
	// where baseWeight + performanceWeight = 1.0
	// This ensures validators are rewarded primarily by power, with performance as an adjustment.
	var totalWeightedPower math.LegacyDec = math.LegacyZeroDec()
	eligibleValidators := make([]types.ValidatorEpochPerformance, 0)

	// Performance weighting: 70% based on power, 30% based on performance
	// This avoids the quadratic penalty while still rewarding good performance
	baseWeight := math.LegacyNewDecWithPrec(70, 2)        // 0.70
	performanceWeight := math.LegacyNewDecWithPrec(30, 2) // 0.30

	for _, performance := range performances {
		// only include validators that meet minimum commit ratio
		if performance.CommitRatio.GTE(params.MinCommitRatio) {
			// Calculate weighted power with performance adjustment
			// Formula: weightedPower = power * (0.7 + 0.3 * commitRatio)
			// This means:
			//   - 100% commit ratio → 1.0x multiplier (full rewards)
			//   -  50% commit ratio → 0.85x multiplier (moderate penalty)
			//   -   0% commit ratio → 0.7x multiplier (but filtered by MinCommitRatio)
			performanceFactor := baseWeight.Add(performanceWeight.Mul(performance.CommitRatio))
			weightedPower := performance.AveragePower.Mul(performanceFactor)
			totalWeightedPower = totalWeightedPower.Add(weightedPower)
			eligibleValidators = append(eligibleValidators, performance)
		}
	}

	if totalWeightedPower.IsZero() {
		feePool, err := k.FeePool.Get(ctx)
		if err != nil {
			return err
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// All fees go directly to validators (no community tax)
	remaining := feesCollected
	feeMultiplier := feesCollected // 100% goes to validators

	// allocate rewards proportionally to performance-weighted power
	validatorsProcessed := 0
	for _, performance := range eligibleValidators {
		// get validator by operator address stored in performance record
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(performance.ValidatorAddress)
		if err != nil {
			k.Logger(ctx).Error("Failed to decode validator address", "error", err)
			continue // Skip this validator but continue with others
		}

		validator, err := k.stakingKeeper.Validator(ctx, valAddr)
		if err != nil {
			k.Logger(ctx).Error("Failed to get validator", "error", err)
			continue // Skip this validator but continue with others
		}

		// calculate reward based on performance-weighted power
		// Use the same performance adjustment formula as above
		performanceFactor := baseWeight.Add(performanceWeight.Mul(performance.CommitRatio))
		weightedPower := performance.AveragePower.Mul(performanceFactor)
		powerFraction := weightedPower.Quo(totalWeightedPower)
		reward := feeMultiplier.MulDecTruncate(powerFraction)

		// Log validator performance metrics and reward calculation
		k.Logger(ctx).Debug("Validator epoch performance and reward calculation",
			"validator", performance.ValidatorAddress,
			"average_power", performance.AveragePower.String(),
			"commit_votes", performance.CommitVotes,
			"total_votes", performance.TotalVotes,
			"commit_ratio", performance.CommitRatio.String(),
			"performance_factor", performanceFactor.String(),
			"weighted_power", weightedPower.String(),
			"power_fraction", powerFraction.String(),
			"reward", reward.String(),
		)

		// Log reward pool calculations for this validator
		k.logRewardPoolCalculations(ctx, validator, reward)

		err = k.AllocateTokensToValidator(ctx, validator, reward)
		if err != nil {
			k.Logger(ctx).Error("Failed to allocate tokens to validator", "validator", validator.GetOperator(), "error", err)
			continue // Continue with other validators
		}

		remaining = remaining.Sub(reward)
		validatorsProcessed++
	}

	// get fee pool and add any remaining to community pool
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		k.Logger(ctx).Error("Failed to set fee pool", "error", err)
		return err
	}

	k.Logger(ctx).Info("Performance-based token allocation completed", "validators_processed", validatorsProcessed, "amount", feesCollectedInt)

	// Log distribution summary
	k.logRewardDistributionSummary(ctx, epochNumber, feesCollectedInt, validatorsProcessed, true)

	// Log multi-validator delegators
	k.logMultiValidatorDelegators(ctx)

	// Log post-distribution validation
	k.logPostDistributionValidation(ctx, epochNumber)

	return nil
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get the validator's NFT and native token shares
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

	// If there are no shares at all, no rewards can be allocated
	if totalShares.IsZero() {
		return nil
	}

	valBz, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	// Get the configurable staking ratios
	nftStakingRatio, err := k.GetNftStakingRatio(ctx)
	if err != nil {
		return err
	}
	nativeStakingRatio, err := k.GetNativeStakingRatio(ctx)
	if err != nil {
		return err
	}

	// Determine how to split the rewards
	var nftRewards, nativeRewards sdk.DecCoins

	k.Logger(ctx).Info(
		"validator reward share snapshot",
		"validator", val.GetOperator(),
		"nft_shares", nftShares.String(),
		"native_shares", nativeShares.String(),
		"total_shares", totalShares.String(),
		"assigned_tokens", tokens.String(),
	)

	// If there are no NFT shares, all rewards go to native token stakers
	if nftShares.IsZero() {
		// All rewards go to native token stakers
		nativeRewards = tokens
		if !tokens.IsZero() {
			k.Logger(ctx).Info(
				"skipping nft reward allocation due to zero nft shares",
				"validator", val.GetOperator(),
				"assigned_tokens", tokens.String(),
			)
		}
	} else if nativeShares.IsZero() {
		// All rewards go to NFT stakers
		nftRewards = tokens
	} else {
		// Split rewards based on the configured ratios
		// Calculate rewards for each type based on the total staking allocation (80%)
		nftRewards = tokens.MulDecTruncate(nftStakingRatio)
		nativeRewards = tokens.MulDecTruncate(nativeStakingRatio)
	}

	// Combine commissions from both reward types
	totalCommission := tokens.MulDec(val.GetCommission())

	// Log validator reward allocation details
	k.logValidatorRewardAllocation(ctx, val, tokens, nftRewards, nativeRewards, totalCommission)

	// Update current commission
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, totalCommission.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator()),
		),
	)
	currentCommission, err := k.GetValidatorAccumulatedCommission(ctx, valBz)
	if err != nil {
		return err
	}

	currentCommission.Commission = currentCommission.Commission.Add(totalCommission...)
	err = k.SetValidatorAccumulatedCommission(ctx, valBz, currentCommission)
	if err != nil {
		return err
	}

	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valBz)
	if err != nil {
		return err
	}

	currentNFTRewards, err := k.GetValidatorCurrentNFTRewards(ctx, valBz)
	if err != nil {
		return err
	}

	currentRewards.Rewards = currentRewards.Rewards.Add(nativeRewards...)
	currentNFTRewards.Rewards = currentNFTRewards.Rewards.Add(nftRewards...)

	err = k.SetValidatorCurrentRewards(ctx, valBz, currentRewards)
	if err != nil {
		return err
	}
	err = k.SetValidatorCurrentNFTRewards(ctx, valBz, currentNFTRewards)
	if err != nil {
		return err
	}

	// Store the split for use during period increment
	// We need to track how much of the current rewards are NFT vs native
	// For this, we'll store metadata about the reward split ratios

	// Update outstanding rewards
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, val.GetOperator()),
		),
	)

	outstanding, err := k.GetValidatorOutstandingRewards(ctx, valBz)
	if err != nil {
		return err
	}

	outstanding.Rewards = outstanding.Rewards.Add(tokens...)
	return k.SetValidatorOutstandingRewards(ctx, valBz, outstanding)
}
