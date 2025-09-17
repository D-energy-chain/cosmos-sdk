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

// withdraw rewards from a delegation
func (k Keeper) WithdrawDelegationRewards(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, types.ErrNoValidatorDistInfo
	}

	var totalRewards sdk.Coins

	// Try to withdraw native delegation rewards first
	del, err := k.stakingKeeper.Delegation(ctx, delAddr, valAddr)
	if err == nil && del != nil {
		// withdraw native delegation rewards
		nativeRewards, err := k.withdrawDelegationRewards(ctx, val, del)
		if err != nil {
			return nil, err
		}
		totalRewards = totalRewards.Add(nativeRewards...)
	}

	// Check for NFT delegations and withdraw NFT rewards
	nftDelegations, err := k.stakingKeeper.GetNFTDelegations(ctx, delAddr, valAddr)
	if err == nil && len(nftDelegations) > 0 {
		// Check if delegator starting info exists for NFT delegations
		hasInfo, err := k.HasDelegatorStartingInfo(ctx, valAddr, delAddr)
		if err != nil {
			return nil, err
		}

		if hasInfo {
			// Get total NFT shares for this delegator with this validator
			totalNFTShares, err := k.stakingKeeper.GetNFTDelegatorShares(ctx, delAddr, valAddr)
			if err != nil {
				return nil, err
			}

			if !totalNFTShares.IsZero() {
				// End current period and calculate NFT rewards
				endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
				if err != nil {
					return nil, err
				}

				// Get delegator starting info to get the starting period and NFT stake
				startingInfo, err := k.GetDelegatorStartingInfo(ctx, valAddr, delAddr)
				if err != nil {
					return nil, err
				}

				startingPeriod := startingInfo.PreviousPeriod
				nftStake := startingInfo.NftStake

				// Calculate NFT delegation rewards using the NFT-specific calculation
				nftRewardsRaw, err := k.calculateNFTDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, nftStake)
				if err != nil {
					return nil, err
				}

				outstanding, err := k.GetValidatorOutstandingRewardsCoins(ctx, valAddr)
				if err != nil {
					return nil, err
				}

				// Defensive edge case handling
				nftRewards := nftRewardsRaw.Intersect(outstanding)
				if !nftRewards.Equal(nftRewardsRaw) {
					logger := k.Logger(ctx)
					logger.Info(
						"rounding error withdrawing NFT rewards from validator",
						"delegator", delAddr.String(),
						"validator", val.GetOperator(),
						"got", nftRewards.String(),
						"expected", nftRewardsRaw.String(),
					)
				}

				// Update outstanding rewards
				outstanding = outstanding.Sub(nftRewards)
				err = k.SetValidatorOutstandingRewards(ctx, valAddr, types.ValidatorOutstandingRewards{Rewards: outstanding})
				if err != nil {
					return nil, err
				}

				// Convert DecCoins to Coins and add to total rewards
				nftRewardsCoins, remainder := nftRewards.TruncateDecimal()
				if !remainder.IsZero() {
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
				totalRewards = totalRewards.Add(nftRewardsCoins...)
			}
		}
	}

	// If no rewards from either native or NFT delegations, return error
	if totalRewards.IsZero() {
		// Check if there's any delegation at all (native or NFT)
		if del == nil && len(nftDelegations) == 0 {
			return nil, types.ErrEmptyDelegationDistInfo
		}
		// No rewards to withdraw, but delegations exist
		return sdk.NewCoins(), nil
	}

	// Transfer rewards to delegator's withdraw address
	withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	// totalRewards is already sdk.Coins, no need to truncate
	finalRewards := totalRewards

	if !finalRewards.IsZero() {
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, finalRewards)
		if err != nil {
			return nil, err
		}
	}

	// reinitialize the delegation tracking (works for both native and NFT)
	err = k.initializeDelegation(ctx, valAddr, delAddr)
	if err != nil {
		return nil, err
	}

	return finalRewards, nil
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
