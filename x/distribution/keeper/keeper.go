package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// Keeper of the distribution store
type Keeper struct {
	storeService  store.KVStoreService
	cdc           codec.BinaryCodec
	authKeeper    types.AccountKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	Schema  collections.Schema
	Params  collections.Item[types.Params]
	FeePool collections.Item[types.FeePool]

	feeCollectorName string // name of the FeeCollector ModuleAccount
}

// NewKeeper creates a new distribution Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, storeService store.KVStoreService,
	ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper,
	feeCollectorName, authority string,
) Keeper {
	// ensure distribution module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService:     storeService,
		cdc:              cdc,
		authKeeper:       ak,
		bankKeeper:       bk,
		stakingKeeper:    sk,
		feeCollectorName: feeCollectorName,
		authority:        authority,
		Params:           collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		FeePool:          collections.NewItem(sb, types.FeePoolKey, "fee_pool", codec.CollValue[types.FeePool](cdc)),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// GetAuthority returns the x/distribution module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With(log.ModuleKey, "x/"+types.ModuleName)
}

// SetWithdrawAddr sets a new address that will receive the rewards upon withdrawal
func (k Keeper) SetWithdrawAddr(ctx context.Context, delegatorAddr, withdrawAddr sdk.AccAddress) error {
	if k.bankKeeper.BlockedAddr(withdrawAddr) {
		return errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive external funds", withdrawAddr)
	}

	withdrawAddrEnabled, err := k.GetWithdrawAddrEnabled(ctx)
	if err != nil {
		return err
	}

	if !withdrawAddrEnabled {
		return types.ErrSetWithdrawAddrDisabled
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeSetWithdrawAddress,
			sdk.NewAttribute(types.AttributeKeyWithdrawAddress, withdrawAddr.String()),
		),
	)

	k.SetDelegatorWithdrawAddr(ctx, delegatorAddr, withdrawAddr)
	return nil
}

// withdraw rewards from a delegation - FIXED VERSION (prevents double period increment and outstanding reward deduction)
func (k Keeper) withdrawDelegationRewards(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	// Fetch the validator
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, types.ErrNoValidatorDistInfo
	}

	// Check if delegator starting info exists (needed for both native and NFT)
	hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	if !hasInfo {
		return nil, types.ErrEmptyDelegationDistInfo
	}

	// Only increment period if there are accumulated rewards that need to be locked in
	// This prevents double period increments when called after epoch end
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	var endingPeriod uint64
	if !currentRewards.Rewards.IsZero() {
		// There are accumulated rewards, increment period to lock them in
		endingPeriod, err = k.IncrementValidatorPeriod(ctx, val)
		if err != nil {
			return nil, err
		}
	} else {
		// No accumulated rewards - rewards already locked in the last completed period
		// Historical rewards only exist for completed periods, not the current active period
		// So we use currentRewards.Period - 1 (the last period that has historical rewards)
		if currentRewards.Period == 0 {
			// Edge case: validator just created, no rewards yet
			endingPeriod = 0
		} else {
			endingPeriod = currentRewards.Period - 1
		}
	}

	// Get delegator starting info
	startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	startingPeriod := startingInfo.PreviousPeriod
	nativeStake := startingInfo.Stake
	nftStake := startingInfo.NftStake

	// Get current outstanding rewards ONCE
	outstanding, err := k.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	var totalRewardsRaw sdk.DecCoins
	var nativeRewardsRaw, nftRewardsRaw sdk.DecCoins

	// Calculate native delegation rewards if they exist
	del, err := k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err == nil && del != nil && !nativeStake.IsZero() {
		nativeRewardsRaw, err = k.calculateDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nativeStake)
		if err != nil {
			return nil, err
		}
		totalRewardsRaw = totalRewardsRaw.Add(nativeRewardsRaw...)
	}

	// Calculate NFT delegation rewards if they exist
	if !nftStake.IsZero() {
		nftRewardsRaw, err = k.calculateNFTDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nftStake)
		if err != nil {
			return nil, err
		}
		totalRewardsRaw = totalRewardsRaw.Add(nftRewardsRaw...)
	}

	// Log detailed calculation breakdown
	k.Logger(ctx).Info("💰 WITHDRAWAL CALCULATION",
		"delegator", delAddr.String(),
		"validator", val.GetOperator(),
		"===== PERIOD INFO =====", "",
		"starting_period", startingPeriod,
		"ending_period", endingPeriod,
		"===== STAKES =====", "",
		"native_stake", nativeStake.String(),
		"nft_stake", nftStake.String(),
		"===== CALCULATED REWARDS (BEFORE INTERSECTION) =====", "",
		"native_rewards_raw", nativeRewardsRaw.String(),
		"nft_rewards_raw", nftRewardsRaw.String(),
		"total_rewards_raw", totalRewardsRaw.String(),
		"===== VALIDATOR STATE =====", "",
		"outstanding_rewards", outstanding.String(),
	)

	// Apply intersection ONCE to total rewards
	totalRewardsDecCoins := totalRewardsRaw.Intersect(outstanding)
	if !totalRewardsDecCoins.Equal(totalRewardsRaw) {
		logger := k.Logger(ctx)
		logger.Info(
			"rounding error withdrawing rewards from validator",
			"delegator", delAddr.String(),
			"validator", val.GetOperator(),
			"got", totalRewardsDecCoins.String(),
			"expected", totalRewardsRaw.String(),
		)
	}

	// Update outstanding rewards ONCE
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
		// add remainder to community pool
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

	// If no rewards, return empty coins (not error if delegations exist)
	if totalRewards.IsZero() {
		return sdk.NewCoins(), nil
	}

	// Transfer rewards to delegator's withdraw address
	withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	// Log BEFORE transfer
	k.Logger(ctx).Info("💰 WITHDRAWAL - BEFORE TRANSFER",
		"delegator", delAddr.String(),
		"validator", val.GetOperator(),
		"withdraw_address", withdrawAddr.String(),
		"amount_to_transfer", totalRewards.String(),
		"remainder_to_community", remainder.String(),
	)

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, totalRewards)
	if err != nil {
		k.Logger(ctx).Error("❌ WITHDRAWAL FAILED - Transfer error",
			"delegator", delAddr.String(),
			"validator", val.GetOperator(),
			"error", err.Error(),
		)
		return nil, err
	}

	// Log AFTER successful transfer
	k.Logger(ctx).Info("✅ WITHDRAWAL SUCCESS",
		"delegator", delAddr.String(),
		"validator", val.GetOperator(),
		"===== BREAKDOWN =====", "",
		"native_rewards", nativeRewardsRaw.String(),
		"nft_rewards", nftRewardsRaw.String(),
		"total_calculated", totalRewardsRaw.String(),
		"after_intersection", totalRewardsDecCoins.String(),
		"===== FINAL TRANSFER =====", "",
		"transferred_to", withdrawAddr.String(),
		"amount_transferred", totalRewards.String(),
		"remainder_to_community", remainder.String(),
		"===== VALIDATOR STATE UPDATE =====", "",
		"new_outstanding_rewards", outstanding.String(),
	)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawRewards,
			sdk.NewAttribute(sdk.AttributeKeyAmount, totalRewards.String()),
			sdk.NewAttribute(types.AttributeKeyValidator, valAddr.String()),
		),
	)

	// reinitialize the delegation tracking (works for both native and NFT)
	err = k.initializeDelegation(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	// Log delegation reward withdrawal
	k.logDelegationRewardWithdrawal(ctx, delAddr, valAddr, totalRewards, "combined")

	return totalRewards, nil
}

// WithdrawDelegationRewards - public API that calls the internal fixed version
func (k Keeper) WithdrawDelegationRewards(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	return k.withdrawDelegationRewards(ctx, delAddr, valAddr)
}

// withdraw validator commission
func (k Keeper) WithdrawValidatorCommission(ctx context.Context, valAddr sdk.ValAddress) (sdk.Coins, error) {
	// fetch validator accumulated commission
	accumCommission, err := k.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if accumCommission.Commission.IsZero() {
		return nil, types.ErrNoValidatorCommission
	}

	commission, remainder := accumCommission.Commission.TruncateDecimal()
	k.SetValidatorAccumulatedCommission(ctx, valAddr, types.ValidatorAccumulatedCommission{Commission: remainder}) // leave remainder to withdraw later

	// update outstanding
	outstanding, err := k.GetValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	err = k.SetValidatorOutstandingRewards(ctx, valAddr, types.ValidatorOutstandingRewards{Rewards: outstanding.Rewards.Sub(sdk.NewDecCoinsFromCoins(commission...))})
	if err != nil {
		return nil, err
	}

	if !commission.IsZero() {
		accAddr := sdk.AccAddress(valAddr)
		withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, accAddr)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, commission)
		if err != nil {
			return nil, err
		}
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeWithdrawCommission,
			sdk.NewAttribute(sdk.AttributeKeyAmount, commission.String()),
		),
	)

	return commission, nil
}

// GetTotalRewards returns the total amount of fee distribution rewards held in the store
func (k Keeper) GetTotalRewards(ctx context.Context) (totalRewards sdk.DecCoins) {
	k.IterateValidatorOutstandingRewards(ctx,
		func(_ sdk.ValAddress, rewards types.ValidatorOutstandingRewards) (stop bool) {
			totalRewards = totalRewards.Add(rewards.Rewards...)
			return false
		},
	)

	return totalRewards
}

// FundCommunityPool allows an account to directly fund the community fund pool.
// The amount is first added to the distribution module account and then directly
// added to the pool. An error is returned if the amount cannot be sent to the
// module account.
func (k Keeper) FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return err
	}

	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(amount...)...)
	return k.FeePool.Set(ctx, feePool)
}

// DelegatorInfo holds information about a delegator for iteration purposes
type DelegatorInfo struct {
	DelegatorAddr sdk.AccAddress
	ValidatorAddr sdk.ValAddress
	StartingInfo  types.DelegatorStartingInfo
}

// IterateValidatorDelegators iterates through all delegators for a specific validator.
// The callback function receives the delegator address and starting info.
// If the callback returns true, iteration stops.
func (k Keeper) IterateValidatorDelegators(
	ctx context.Context,
	valAddr sdk.ValAddress,
	callback func(delAddr sdk.AccAddress, startingInfo types.DelegatorStartingInfo) (stop bool),
) error {
	// Iterate through all delegator starting infos and filter by validator
	k.IterateDelegatorStartingInfos(ctx, func(iterValAddr sdk.ValAddress, delAddr sdk.AccAddress, info types.DelegatorStartingInfo) (stop bool) {
		// Only process delegators for the specified validator
		if iterValAddr.Equals(valAddr) {
			return callback(delAddr, info)
		}
		return false
	})
	return nil
}

// IterateAllValidatorsAndDelegators iterates through all validators and their delegators.
// The callback function receives delegator info for each delegator of each validator.
// If the callback returns true, iteration stops.
func (k Keeper) IterateAllValidatorsAndDelegators(
	ctx context.Context,
	callback func(info DelegatorInfo) (stop bool),
) error {
	// Iterate through all delegator starting infos
	// This is more efficient than iterating validators then delegators for each
	k.IterateDelegatorStartingInfos(ctx, func(valAddr sdk.ValAddress, delAddr sdk.AccAddress, startingInfo types.DelegatorStartingInfo) (stop bool) {
		info := DelegatorInfo{
			DelegatorAddr: delAddr,
			ValidatorAddr: valAddr,
			StartingInfo:  startingInfo,
		}
		return callback(info)
	})
	return nil
}
