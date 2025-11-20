package keeper

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

const epochIdentifier = types.DefaultDelegationEpochIdentifier

// NextQueuedDelegationID increments and returns the next queued delegation identifier.
func (k Keeper) NextQueuedDelegationID(ctx context.Context) (uint64, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.QueuedDelegationIDKey)
	if err != nil {
		return 0, err
	}

	var id uint64
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}

	id++
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, id)
	if err := store.Set(types.QueuedDelegationIDKey, buf); err != nil {
		return 0, err
	}

	return id, nil
}

// getQueuedDelegation fetches the queued delegation entry for the given pair.
func (k Keeper) getQueuedDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (types.QueuedDelegation, error) {
	return k.queuedDelegations.Get(ctx, collections.Join(delAddr, valAddr))
}

// setQueuedDelegation stores the queued delegation entry and updates the validator index.
func (k Keeper) setQueuedDelegation(ctx context.Context, qd types.QueuedDelegation) error {
	delegatorAddr, err := k.authKeeper.AddressCodec().StringToBytes(qd.DelegatorAddress)
	if err != nil {
		return err
	}
	validatorAddr, err := k.validatorAddressCodec.StringToBytes(qd.ValidatorAddress)
	if err != nil {
		return err
	}

	del := sdk.AccAddress(delegatorAddr)
	val := sdk.ValAddress(validatorAddr)

	if err := k.queuedDelegations.Set(ctx, collections.Join(del, val), qd); err != nil {
		return err
	}

	return k.queuedDelegationByValIdx.Set(ctx, collections.Join(val, del), []byte{})
}

// removeQueuedDelegation removes the queued delegation entry and cleans the validator index.
func (k Keeper) removeQueuedDelegation(ctx context.Context, qd types.QueuedDelegation) error {
	delegatorAddr, err := k.authKeeper.AddressCodec().StringToBytes(qd.DelegatorAddress)
	if err != nil {
		return err
	}
	validatorAddr, err := k.validatorAddressCodec.StringToBytes(qd.ValidatorAddress)
	if err != nil {
		return err
	}

	del := sdk.AccAddress(delegatorAddr)
	val := sdk.ValAddress(validatorAddr)

	if err := k.queuedDelegations.Remove(ctx, collections.Join(del, val)); err != nil {
		return err
	}

	return k.queuedDelegationByValIdx.Remove(ctx, collections.Join(val, del))
}

// hasMaxQueuedDelegationEntries checks if the queued delegations reached the configured maximum.
func (k Keeper) hasMaxQueuedDelegationEntries(ctx context.Context, delegatorAddr sdk.AccAddress, validatorAddr sdk.ValAddress) (bool, error) {
	qd, err := k.getQueuedDelegation(ctx, delegatorAddr, validatorAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	maxEntries, err := k.MaxEntries(ctx)
	if err != nil {
		return false, err
	}

	return len(qd.Entries) >= int(maxEntries), nil
}

// getDelegationQueueEpoch retrieves queued delegation DVPairs for the provided epoch.
func (k Keeper) getDelegationQueueEpoch(ctx context.Context, epoch sdkmath.Int) ([]types.DVPair, error) {
	queue, err := k.queuedDelegationQueue.Get(ctx, epoch.Int64())
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return []types.DVPair{}, nil
		}
		return nil, err
	}

	return queue.Pairs, nil
}

// setDelegationQueueEpoch stores the queue DVPairs for a specific epoch.
func (k Keeper) setDelegationQueueEpoch(ctx context.Context, epoch sdkmath.Int, pairs []types.DVPair) error {
	return k.queuedDelegationQueue.Set(ctx, epoch.Int64(), types.DVPairs{Pairs: pairs})
}

// insertDelegationQueue adds a delegator/validator pair to the activation queue for the provided epoch.
func (k Keeper) insertDelegationQueue(ctx context.Context, qd types.QueuedDelegation, activationEpoch sdkmath.Int) error {
	dvPair := types.DVPair{
		DelegatorAddress: qd.DelegatorAddress,
		ValidatorAddress: qd.ValidatorAddress,
	}

	queue, err := k.getDelegationQueueEpoch(ctx, activationEpoch)
	if err != nil {
		return err
	}

	if len(queue) == 0 {
		return k.setDelegationQueueEpoch(ctx, activationEpoch, []types.DVPair{dvPair})
	}

	queue = append(queue, dvPair)
	return k.setDelegationQueueEpoch(ctx, activationEpoch, queue)
}

// dequeueAllMatureDelegationQueue returns all queued delegations with activation epoch <= currEpoch and removes them from the queue.
func (k Keeper) dequeueAllMatureDelegationQueue(ctx context.Context, currEpoch sdkmath.Int) ([]types.DVPair, error) {
	iter, err := k.queuedDelegationQueue.Iterate(ctx, (&collections.Range[int64]{}).EndInclusive(currEpoch.Int64()))
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	matured := make([]types.DVPair, 0)

	for ; iter.Valid(); iter.Next() {
		pairs, err := iter.Value()
		if err != nil {
			return nil, err
		}

		matured = append(matured, pairs.Pairs...)
		key, err := iter.Key()
		if err != nil {
			return nil, err
		}

		if err := k.queuedDelegationQueue.Remove(ctx, key); err != nil {
			return nil, err
		}
	}

	return matured, nil
}

// queueDelegation enqueues a staking delegation for activation in the next epoch.
func (k Keeper) queueDelegation(ctx context.Context, delegatorAddr sdk.AccAddress, validator types.Validator, amount sdk.Coin) (types.DelegationQueueEntry, sdkmath.Int, error) {
	if k.epochKeeper == nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, fmt.Errorf("epoch keeper not configured")
	}

	valAddrBz, err := k.validatorAddressCodec.StringToBytes(validator.GetOperator())
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	hasMax, err := k.hasMaxQueuedDelegationEntries(ctx, delegatorAddr, sdk.ValAddress(valAddrBz))
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	if hasMax {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, types.ErrMaxQueuedDelegationEntries
	}

	poolName := types.NotBondedPoolName
	bondStatus := types.Unbonded
	switch {
	case validator.IsBonded():
		poolName = types.BondedPoolName
		bondStatus = types.Bonded
	case validator.IsUnbonding(), validator.IsUnbonded():
		// keep defaults
	default:
		return types.DelegationQueueEntry{}, sdkmath.Int{}, fmt.Errorf("invalid validator status: %d", validator.Status)
	}

	currentEpoch, found := k.getCurrentEpoch(ctx)
	if !found {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, fmt.Errorf("unable to determine current epoch")
	}

	activationEpoch := currentEpoch.Add(sdkmath.NewInt(types.EpochIncrementOffset))

	entryID, err := k.NextQueuedDelegationID(ctx)
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	coins := sdk.NewCoins(amount)
	if err := k.bankKeeper.DelegateCoinsFromAccountToModule(ctx, delegatorAddr, poolName, coins); err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	entry := types.DelegationQueueEntry{
		Id:              entryID,
		CreationEpoch:   currentEpoch,
		ActivationEpoch: activationEpoch,
		Amount:          amount.Amount,
		Denom:           amount.Denom,
		BondStatus:      bondStatus,
	}

	delStr, err := k.authKeeper.AddressCodec().BytesToString(delegatorAddr)
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	queued, err := k.getQueuedDelegation(ctx, delegatorAddr, sdk.ValAddress(valAddrBz))
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			queued = types.QueuedDelegation{
				DelegatorAddress: delStr,
				ValidatorAddress: validator.GetOperator(),
				Entries:          []types.DelegationQueueEntry{entry},
			}
		} else {
			return types.DelegationQueueEntry{}, sdkmath.Int{}, err
		}
	} else {
		queued.Entries = append(queued.Entries, entry)
	}

	if err := k.setQueuedDelegation(ctx, queued); err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	if err := k.insertDelegationQueue(ctx, queued, activationEpoch); err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, err
	}

	return entry, activationEpoch, nil
}

// ProcessQueuedDelegations activates all queued delegations scheduled up to the next epoch.
func (k Keeper) ProcessQueuedDelegations(ctx context.Context, epoch int64) error {
	if k.epochKeeper == nil {
		return nil
	}

	effectiveEpoch := sdkmath.NewInt(epoch).Add(sdkmath.NewInt(types.EpochIncrementOffset))
	maturedPairs, err := k.dequeueAllMatureDelegationQueue(ctx, effectiveEpoch)
	if err != nil {
		return err
	}

	for _, pair := range maturedPairs {
		if err := k.processQueuedDelegationsForPair(ctx, effectiveEpoch, pair.DelegatorAddress, pair.ValidatorAddress); err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) processQueuedDelegationsForPair(ctx context.Context, currentEpoch sdkmath.Int, delegatorAddrStr, validatorAddrStr string) error {
	delegatorAddr, err := k.authKeeper.AddressCodec().StringToBytes(delegatorAddrStr)
	if err != nil {
		return err
	}

	validatorAddr, err := k.validatorAddressCodec.StringToBytes(validatorAddrStr)
	if err != nil {
		return err
	}

	del := sdk.AccAddress(delegatorAddr)
	val := sdk.ValAddress(validatorAddr)

	queued, err := k.getQueuedDelegation(ctx, del, val)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil
		}
		return err
	}

	updated := false
	for i := 0; i < len(queued.Entries); {
		entry := queued.Entries[i]
		if entry.ActivationEpoch.LTE(currentEpoch) {
			if err := k.activateQueuedDelegationEntry(ctx, del, validatorAddrStr, entry); err != nil {
				return err
			}
			queued.Entries = append(queued.Entries[:i], queued.Entries[i+1:]...)
			updated = true
		} else {
			i++
		}
	}

	if !updated {
		return nil
	}

	if len(queued.Entries) == 0 {
		return k.removeQueuedDelegation(ctx, queued)
	}

	return k.setQueuedDelegation(ctx, queued)
}

func (k Keeper) activateQueuedDelegationEntry(ctx context.Context, delegatorAddr sdk.AccAddress, validatorAddrStr string, entry types.DelegationQueueEntry) error {
	validatorAddr, err := k.validatorAddressCodec.StringToBytes(validatorAddrStr)
	if err != nil {
		return err
	}

	validator, err := k.GetValidator(ctx, validatorAddr)
	if err != nil {
		return err
	}

	denom, err := k.BondDenom(ctx)
	if err != nil {
		return err
	}

	if entry.Denom != denom {
		return fmt.Errorf("queued delegation denom mismatch: expected %s got %s", denom, entry.Denom)
	}

	bondStatus := types.BondStatus(entry.BondStatus)
	newShares, err := k.Delegate(ctx, delegatorAddr, entry.Amount, bondStatus, validator, false)
	if err != nil {
		return err
	}

	event := sdk.NewEvent(
		types.EventTypeDelegate,
		sdk.NewAttribute(types.AttributeKeyValidator, validatorAddrStr),
		sdk.NewAttribute(types.AttributeKeyDelegator, delegatorAddr.String()),
		sdk.NewAttribute(sdk.AttributeKeyAmount, sdk.NewCoin(denom, entry.Amount).String()),
		sdk.NewAttribute(types.AttributeKeyQueueEntryID, fmt.Sprintf("%d", entry.Id)),
		sdk.NewAttribute(types.AttributeKeyActivationEpoch, entry.ActivationEpoch.String()),
		sdk.NewAttribute(types.AttributeKeyNewShares, newShares.String()),
	)
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(event)

	return nil
}

// CancelQueuedDelegation removes a queued delegation entry and refunds the escrowed tokens.
func (k Keeper) CancelQueuedDelegation(ctx context.Context, delegator sdk.AccAddress, entryID uint64) (types.DelegationQueueEntry, sdk.ValAddress, error) {
	var (
		targetEntry types.DelegationQueueEntry
		parent      types.QueuedDelegation
		found       bool
	)

	rng := collections.NewPrefixedPairRange[sdk.AccAddress, sdk.ValAddress](delegator)

	err := k.queuedDelegations.Walk(
		ctx,
		rng,
		func(_ collections.Pair[sdk.AccAddress, sdk.ValAddress], value types.QueuedDelegation) (bool, error) {
			for idx, entry := range value.Entries {
				if entry.Id == entryID {
					targetEntry = entry
					value.Entries = append(value.Entries[:idx], value.Entries[idx+1:]...)
					parent = value
					found = true
					return true, nil
				}
			}
			return false, nil
		},
	)
	if err != nil {
		return types.DelegationQueueEntry{}, nil, err
	}

	if !found {
		return types.DelegationQueueEntry{}, nil, types.ErrQueuedDelegationNotFound
	}

	if len(parent.Entries) == 0 {
		if err := k.removeQueuedDelegation(ctx, parent); err != nil {
			return types.DelegationQueueEntry{}, nil, err
		}
	} else {
		if err := k.setQueuedDelegation(ctx, parent); err != nil {
			return types.DelegationQueueEntry{}, nil, err
		}
	}

	valAddrBz, err := k.validatorAddressCodec.StringToBytes(parent.ValidatorAddress)
	if err != nil {
		return types.DelegationQueueEntry{}, nil, err
	}

	if err := k.processQueuedDelegationCancellation(ctx, delegator, targetEntry); err != nil {
		return types.DelegationQueueEntry{}, nil, err
	}

	return targetEntry, sdk.ValAddress(valAddrBz), nil
}

func moduleNameForBondStatus(status types.BondStatus) string {
	switch status {
	case types.Bonded:
		return types.BondedPoolName
	default:
		return types.NotBondedPoolName
	}
}

func (k Keeper) getCurrentEpoch(ctx context.Context) (sdkmath.Int, bool) {
	if k.epochKeeper == nil {
		return sdkmath.Int{}, false
	}
	return math.NewInt(k.epochKeeper.GetCurrentEpochNumber(sdk.UnwrapSDKContext(ctx), epochIdentifier)), true
}

// QueuedDelegationsEnabled returns whether queued delegations are active.
func (k Keeper) QueuedDelegationsEnabled(ctx context.Context) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	return params.EnableQueuedDelegations, nil
}

// queueDelegationOrDelegate handles delegation based on the queued delegation setting.
func (k Keeper) queueDelegationOrDelegate(
	ctx context.Context,
	delegator sdk.AccAddress,
	validator types.Validator,
	amount sdk.Coin,
) (types.DelegationQueueEntry, sdkmath.Int, bool, error) {
	enabled, err := k.QueuedDelegationsEnabled(ctx)
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, false, err
	}

	if !enabled {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, false, nil
	}

	entry, activationEpoch, err := k.queueDelegation(ctx, delegator, validator, amount)
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, false, err
	}

	return entry, activationEpoch, true, nil
}

// processQueuedDelegationCancellation refunds coins from the correct pool when canceling queued delegations.
func (k Keeper) processQueuedDelegationCancellation(ctx context.Context, delegator sdk.AccAddress, entry types.DelegationQueueEntry) error {
	moduleName := moduleNameForBondStatus(types.BondStatus(entry.BondStatus))
	coins := sdk.NewCoins(sdk.NewCoin(entry.Denom, entry.Amount))
	return k.bankKeeper.UndelegateCoinsFromModuleToAccount(ctx, moduleName, delegator, coins)
}

// handleImmediateDelegation executes a direct delegation without queueing.
func (k Keeper) handleImmediateDelegation(ctx context.Context, delegator sdk.AccAddress, validator types.Validator, amount sdk.Coin) (sdkmath.LegacyDec, error) {
	newShares, err := k.Delegate(ctx, delegator, amount.Amount, types.Unbonded, validator, true)
	if err != nil {
		return sdkmath.LegacyDec{}, err
	}
	return newShares, nil
}

// EnqueueOrDelegate wraps queueing logic and returns whether queueing occurred.
func (k Keeper) EnqueueOrDelegate(
	ctx context.Context,
	delegator sdk.AccAddress,
	validator types.Validator,
	amount sdk.Coin,
) (types.DelegationQueueEntry, sdkmath.Int, sdkmath.LegacyDec, bool, error) {
	entry, epoch, queued, err := k.queueDelegationOrDelegate(ctx, delegator, validator, amount)
	if err != nil {
		return types.DelegationQueueEntry{}, sdkmath.Int{}, sdkmath.LegacyDec{}, false, err
	}

	if queued {
		return entry, epoch, sdkmath.LegacyDec{}, true, nil
	}

	newShares, err := k.handleImmediateDelegation(ctx, delegator, validator, amount)
	return entry, epoch, newShares, false, err
}

func wrapDelegationError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, types.ErrMaxQueuedDelegationEntries),
		errors.Is(err, types.ErrQueuedDelegationsDisabled),
		errors.Is(err, types.ErrQueuedDelegationNotFound):
		return err
	default:
		return cosmossdkerrors.Wrap(sdkerrors.ErrInvalidRequest, err.Error())
	}
}
