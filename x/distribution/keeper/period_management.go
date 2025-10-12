package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// ensurePeriod0Exists checks if period 0 exists for a validator, and creates it if missing.
// This handles backward compatibility for validators created before period 0 tracking was implemented,
// genesis validators, and chain upgrades.
//
// Period 0 represents the genesis/starting state with zero cumulative rewards.
// It MUST exist for reward calculations to work correctly.
func (k Keeper) ensurePeriod0Exists(ctx context.Context, valAddr sdk.ValAddress) error {
	// Try to get period 0
	_, err := k.GetValidatorHistoricalRewards(ctx, valAddr, 0)
	if err == nil {
		// Period 0 exists, nothing to do
		return nil
	}

	// Period 0 doesn't exist - create it now (lazy initialization)
	initHeight := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	
	// Create period 0 with zero cumulative rewards
	// This represents "no rewards earned yet" which is correct for the starting state
	period0 := types.ValidatorHistoricalRewards{
		CumulativeRewardRatio:    sdk.NewDecCoins(),
		NftCumulativeRewardRatio: sdk.NewDecCoins(),
		ReferenceCount:           1,
		Height:                   initHeight,
	}

	err = k.SetValidatorHistoricalRewards(ctx, valAddr, 0, period0)
	if err != nil {
		return err
	}

	k.Logger(ctx).Info("📌 Lazy-initialized period 0 for validator (backward compatibility)",
		"validator", valAddr.String(),
		"reason", "period_0_missing",
		"height", initHeight,
	)

	return nil
}

