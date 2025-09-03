package keeper

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// SetValidatorEpochPerformance sets a validator's epoch performance record.
func (k Keeper) SetValidatorEpochPerformance(ctx context.Context, valAddr sdk.ValAddress, performance types.ValidatorEpochPerformance) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetValidatorEpochPerformanceKey(valAddr, performance.EpochIdentifier, performance.EpochNumber)
	bz, err := k.cdc.Marshal(&performance)
	if err != nil {
		return err
	}
	return store.Set(key, bz)
}

// GetValidatorEpochPerformance retrieves a validator's epoch performance record.
func (k Keeper) GetValidatorEpochPerformance(ctx context.Context, valAddr sdk.ValAddress, epochIdentifier string, epochNumber int64) (types.ValidatorEpochPerformance, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetValidatorEpochPerformanceKey(valAddr, epochIdentifier, epochNumber)

	bz, err := store.Get(key)
	if err != nil {
		return types.ValidatorEpochPerformance{}, err
	}
	if bz == nil {
		return types.ValidatorEpochPerformance{}, types.ErrNoValidatorDistInfo
	}

	var performance types.ValidatorEpochPerformance
	if err := k.cdc.Unmarshal(bz, &performance); err != nil {
		return types.ValidatorEpochPerformance{}, err
	}

	return performance, nil
}

// HasValidatorEpochPerformance checks if a validator has a performance record for a specific epoch.
func (k Keeper) HasValidatorEpochPerformance(ctx context.Context, valAddr sdk.ValAddress, epochIdentifier string, epochNumber int64) (bool, error) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetValidatorEpochPerformanceKey(valAddr, epochIdentifier, epochNumber)
	return store.Has(key)
}

// DeleteValidatorEpochPerformance deletes a validator's epoch performance record.
func (k Keeper) DeleteValidatorEpochPerformance(ctx context.Context, valAddr sdk.ValAddress, epochIdentifier string, epochNumber int64) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.GetValidatorEpochPerformanceKey(valAddr, epochIdentifier, epochNumber)
	return store.Delete(key)
}

// UpdateValidatorEpochPerformance updates or creates a validator's epoch performance record.
// This method should be called during EndBlock to track validator commit votes.
func (k Keeper) UpdateValidatorEpochPerformance(ctx context.Context, valAddr sdk.ValAddress, epochIdentifier string, epochNumber int64, power int64, committed bool) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Get existing performance record or create new one
	performance, err := k.GetValidatorEpochPerformance(ctx, valAddr, epochIdentifier, epochNumber)
	if err != nil && err != types.ErrNoValidatorDistInfo {
		return err
	}

	// Initialize if new record
	if err == types.ErrNoValidatorDistInfo {
		performance = types.ValidatorEpochPerformance{
			ValidatorAddress: valAddr.String(),
			EpochIdentifier:  epochIdentifier,
			EpochNumber:      epochNumber,
			CommitVotes:      0,
			TotalVotes:       0,
			CommitRatio:      math.LegacyZeroDec(),
			AveragePower:     math.LegacyZeroDec(),
		}
	}

	// Update vote counts
	performance.TotalVotes++
	if committed {
		performance.CommitVotes++
	}

	// Update average power (running average)
	if performance.TotalVotes == 1 {
		performance.AveragePower = math.LegacyNewDec(power)
	} else {
		// Running average: new_avg = old_avg + (new_value - old_avg) / count
		oldAvg := performance.AveragePower
		newValue := math.LegacyNewDec(power)
		count := math.LegacyNewDec(performance.TotalVotes)
		performance.AveragePower = oldAvg.Add(newValue.Sub(oldAvg).Quo(count))
	}

	// Calculate commit ratio
	if performance.TotalVotes > 0 {
		performance.CommitRatio = math.LegacyNewDec(performance.CommitVotes).Quo(math.LegacyNewDec(performance.TotalVotes))
	}

	sdkCtx.Logger().Debug("Updated validator epoch performance",
		"validator", valAddr.String(),
		"epoch_identifier", epochIdentifier,
		"epoch_number", epochNumber,
		"commit_votes", performance.CommitVotes,
		"total_votes", performance.TotalVotes,
		"commit_ratio", performance.CommitRatio,
		"average_power", performance.AveragePower)

	return k.SetValidatorEpochPerformance(ctx, valAddr, performance)
}

// IterateValidatorEpochPerformanceByValidator iterates over all epoch performance records for a specific validator.
func (k Keeper) IterateValidatorEpochPerformanceByValidator(ctx context.Context, valAddr sdk.ValAddress, fn func(performance types.ValidatorEpochPerformance) (stop bool)) error {
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	prefix := types.GetValidatorEpochPerformancePrefix(valAddr)

	iterator := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var performance types.ValidatorEpochPerformance
		if err := k.cdc.Unmarshal(iterator.Value(), &performance); err != nil {
			return err
		}

		if fn(performance) {
			break
		}
	}

	return nil
}

// GetAllValidatorEpochPerformanceForEpoch retrieves all validator performance records for a specific epoch.
func (k Keeper) GetAllValidatorEpochPerformanceForEpoch(ctx context.Context, epochIdentifier string, epochNumber int64) ([]types.ValidatorEpochPerformance, error) {
	var performances []types.ValidatorEpochPerformance

	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := store.Iterator(types.ValidatorEpochPerformancePrefix, storetypes.PrefixEndBytes(types.ValidatorEpochPerformancePrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var performance types.ValidatorEpochPerformance
		if err := k.cdc.Unmarshal(iterator.Value(), &performance); err != nil {
			return nil, err
		}

		// Filter by epoch identifier and number
		if performance.EpochIdentifier == epochIdentifier && performance.EpochNumber == epochNumber {
			performances = append(performances, performance)
		}
	}

	return performances, nil
}

// CleanupOldEpochPerformanceRecords removes epoch performance records older than a specified number of epochs.
// This helps prevent unbounded state growth.
func (k Keeper) CleanupOldEpochPerformanceRecords(ctx context.Context, epochIdentifier string, currentEpochNumber int64, keepLastN int64) error {
	cutoffEpoch := currentEpochNumber - keepLastN
	if cutoffEpoch <= 0 {
		return nil // Nothing to clean up
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := runtime.KVStoreAdapter(k.storeService.OpenKVStore(ctx))
	iterator := store.Iterator(types.ValidatorEpochPerformancePrefix, storetypes.PrefixEndBytes(types.ValidatorEpochPerformancePrefix))
	defer iterator.Close()

	keysToDelete := make([][]byte, 0)

	for ; iterator.Valid(); iterator.Next() {
		var performance types.ValidatorEpochPerformance
		if err := k.cdc.Unmarshal(iterator.Value(), &performance); err != nil {
			continue // Skip malformed records
		}

		// Delete records older than cutoff for this epoch identifier
		if performance.EpochIdentifier == epochIdentifier && performance.EpochNumber < cutoffEpoch {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
	}

	// Delete the collected keys
	for _, key := range keysToDelete {
		store.Delete(key)
	}

	sdkCtx.Logger().Info("Cleaned up old epoch performance records",
		"epoch_identifier", epochIdentifier,
		"current_epoch", currentEpochNumber,
		"cutoff_epoch", cutoffEpoch,
		"records_deleted", len(keysToDelete))

	return nil
}

// TrackValidatorPerformance can be called directly by parent modules to track validator performance
// This avoids polluting shared hook interfaces with module-specific methods
func (k Keeper) TrackValidatorPerformance(ctx sdk.Context, epochIdentifier string, epochNumber int64, voteInfos []abci.VoteInfo) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	
	// get distribution parameters to check if performance-based distribution is enabled
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	// Only track performance if performance-based distribution is enabled
	if !params.EnablePerformanceBasedDistribution {
		return nil
	}

	// Track validator performance for each vote in this block
	for _, voteInfo := range voteInfos {
		// Get validator by consensus address
		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, voteInfo.Validator.Address)
		if err != nil {
			sdkCtx.Logger().Debug("Failed to get validator by consensus address", 
				"cons_addr", voteInfo.Validator.Address, 
				"error", err)
			continue // Skip this validator but continue with others
		}

		// Determine if this was a commit vote
		committed := voteInfo.BlockIdFlag == cmtproto.BlockIDFlagCommit

		// Convert validator operator address to ValAddress
		valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(validator.GetOperator())
		if err != nil {
			sdkCtx.Logger().Error("Failed to convert validator operator address",
				"operator", validator.GetOperator(),
				"error", err)
			continue
		}

		// Update the validator's epoch performance
		err = k.UpdateValidatorEpochPerformance(
			ctx, 
			valAddr,
			epochIdentifier,
			epochNumber,
			voteInfo.Validator.Power,
			committed,
		)
		if err != nil {
			sdkCtx.Logger().Error("Failed to update validator epoch performance",
				"validator", validator.GetOperator(),
				"epoch_identifier", epochIdentifier,
				"epoch_number", epochNumber,
				"error", err)
			continue
		}

		sdkCtx.Logger().Debug("Updated validator epoch performance",
			"validator", validator.GetOperator(),
			"epoch_identifier", epochIdentifier,
			"epoch_number", epochNumber,
			"power", voteInfo.Validator.Power,
			"committed", committed,
			"block_height", sdkCtx.BlockHeight())
	}

	return nil
}
