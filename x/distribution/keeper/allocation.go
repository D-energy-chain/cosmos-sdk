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
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("AllocateTokens started", 
		"total_previous_power", totalPreviousPower, 
		"num_bonded_votes", len(bondedVotes))

	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	sdkCtx.Logger().Info("Fees collected from fee collector", 
		"fees_collected_int", feesCollectedInt, 
		"fees_collected_dec", feesCollected)

	if feesCollectedInt.IsZero() {
		sdkCtx.Logger().Warn("No fees collected - skipping token allocation")
		return nil
	}

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		sdkCtx.Logger().Error("Failed to transfer fees to distribution module", "error", err)
		return err
	}

	sdkCtx.Logger().Info("Fees transferred to distribution module", 
		"from", k.feeCollectorName, 
		"to", types.ModuleName, 
		"amount", feesCollectedInt)

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	if totalPreviousPower == 0 {
		sdkCtx.Logger().Warn("Total previous power is zero - allocating all fees to community pool")
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// calculate fraction allocated to validators
	remaining := feesCollected
	communityTax, err := k.GetCommunityTax(ctx)
	if err != nil {
		return err
	}

	sdkCtx.Logger().Info("Community tax configuration", 
		"community_tax", communityTax)

	voteMultiplier := math.LegacyOneDec().Sub(communityTax)
	feeMultiplier := feesCollected.MulDecTruncate(voteMultiplier)

	sdkCtx.Logger().Info("Fee distribution calculations", 
		"remaining", remaining, 
		"vote_multiplier", voteMultiplier, 
		"fee_multiplier", feeMultiplier)

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
			sdkCtx.Logger().Debug("Skipping validator that missed vote", 
				"validator_address", vote.Validator.Address, 
				"block_id_flag", vote.BlockIdFlag)
			continue
		}

		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		if err != nil {
			sdkCtx.Logger().Error("Failed to get validator by consensus address", 
				"cons_addr", vote.Validator.Address, 
				"error", err)
			return err
		}

		powerFraction := math.LegacyNewDec(vote.Validator.Power).QuoTruncate(math.LegacyNewDec(totalPreviousPower))
		reward := feeMultiplier.MulDecTruncate(powerFraction)

		sdkCtx.Logger().Info("Allocating tokens to validator", 
			"validator", validator.GetOperator(), 
			"power", vote.Validator.Power, 
			"power_fraction", powerFraction, 
			"reward", reward)

		err = k.AllocateTokensToValidator(ctx, validator, reward)
		if err != nil {
			sdkCtx.Logger().Error("Failed to allocate tokens to validator", 
				"validator", validator.GetOperator(), 
				"error", err)
			return err
		}

		remaining = remaining.Sub(reward)
		validatorsProcessed++
	}

	sdkCtx.Logger().Info("Validator reward allocation completed", 
		"validators_processed", validatorsProcessed, 
		"remaining_for_community", remaining)

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		sdkCtx.Logger().Error("Failed to set fee pool", "error", err)
		return err
	}

	sdkCtx.Logger().Info("Token allocation completed successfully", 
		"community_pool_addition", remaining)
	return nil
}

// AllocateTokensToValidator allocate tokens to a particular validator,
// splitting according to commission.
func (k Keeper) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	
	// Get the validator's NFT and native token shares
	nftShares := val.GetNFTDelegatorShares()
	nativeShares := val.GetDelegatorShares()
	totalShares := nftShares.Add(nativeShares)

	sdkCtx.Logger().Info("AllocateTokensToValidator started", 
		"validator", val.GetOperator(), 
		"tokens", tokens, 
		"nft_shares", nftShares, 
		"native_shares", nativeShares, 
		"total_shares", totalShares)

	// If there are no shares at all, no rewards can be allocated
	if totalShares.IsZero() {
		sdkCtx.Logger().Warn("No shares for validator - skipping reward allocation", 
			"validator", val.GetOperator())
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
	var nftCommission, nativeCommission sdk.DecCoins
	var nftShared, nativeShared sdk.DecCoins

	// If there are no NFT shares, all rewards go to native token stakers
	if nftShares.IsZero() {
		// All rewards go to native token stakers
		nativeRewards = tokens
		nativeCommission = tokens.MulDec(val.GetCommission())
		nativeShared = tokens.Sub(nativeCommission)
	} else if nativeShares.IsZero() {
		// All rewards go to NFT stakers
		nftRewards = tokens
		nftCommission = tokens.MulDec(val.GetCommission())
		nftShared = tokens.Sub(nftCommission)
	} else {
		// Split rewards based on the configured ratios
		// Calculate rewards for each type based on the total staking allocation (80%)
		nftRewards = tokens.MulDecTruncate(nftStakingRatio)
		nativeRewards = tokens.MulDecTruncate(nativeStakingRatio)

		// Apply commission to each reward type
		nftCommission = nftRewards.MulDec(val.GetCommission())
		nftShared = nftRewards.Sub(nftCommission)

		nativeCommission = nativeRewards.MulDec(val.GetCommission())
		nativeShared = nativeRewards.Sub(nativeCommission)
	}

	// Combine commissions from both reward types
	totalCommission := nftCommission.Add(nativeCommission...)

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

	// Update current rewards - combine both native and NFT rewards
	totalShared := nftShared.Add(nativeShared...)
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valBz)
	if err != nil {
		return err
	}

	currentRewards.Rewards = currentRewards.Rewards.Add(totalShared...)
	err = k.SetValidatorCurrentRewards(ctx, valBz, currentRewards)
	if err != nil {
		return err
	}

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
