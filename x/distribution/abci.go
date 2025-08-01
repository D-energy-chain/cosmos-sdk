package distribution

import (
	"fmt"
	
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/exported"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// BeginBlocker sets the proposer for determining distribution during endblock
// and accumulates rewards for the current block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper, ek exported.EpochKeeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, telemetry.Now(), telemetry.MetricKeyBeginBlocker)

	ctx.Logger().Info("Distribution BeginBlocker started", 
		"height", ctx.BlockHeight(), 
		"time", ctx.BlockTime(),
		"epoch_keeper_type", fmt.Sprintf("%T", ek))

	// determine the total power signing the block
	var previousTotalPower int64
	for _, voteInfo := range ctx.VoteInfos() {
		previousTotalPower += voteInfo.Validator.Power
	}

	ctx.Logger().Info("Vote info processed", 
		"total_power", previousTotalPower, 
		"num_validators", len(ctx.VoteInfos()))

	// record the proposer for when we payout on the next block
	if err := SetProposerFromBlockHeader(ctx, k); err != nil {
		ctx.Logger().Error("Failed to set proposer from block header", "error", err)
		return err
	}

	// Check if the configured epoch has ended. The identifier is defined in
	// types.DefaultEpochIdentifier so it can be configured from one place.
	isEpochEnd := ek.IsEpochEnd(ctx, types.DefaultEpochIdentifier)
	ctx.Logger().Info("Checking epoch end status", 
		"epoch_identifier", types.DefaultEpochIdentifier, 
		"is_epoch_end", isEpochEnd)

	if isEpochEnd {
		ctx.Logger().Info("Epoch ended - starting reward distribution", 
			"total_power", previousTotalPower)

		// Note: We can't easily get the fee collector balance here because
		// the keeper methods are private, but AllocateTokens will log this information

		// At epoch end, distribute all accumulated rewards
		if err := k.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos()); err != nil {
			ctx.Logger().Error("Failed to allocate tokens during epoch end", "error", err)
			return err
		}

		// AllocateTokens function contains detailed logging of the fee collection process

		ctx.Logger().Info("Epoch-based reward distribution completed successfully")
		return nil
	}

	ctx.Logger().Info("Not at epoch end - skipping reward distribution", 
		"epoch_identifier", types.DefaultEpochIdentifier)

	// If not at epoch end, just collect fees but don't distribute them
	// The fees will remain in the fee collector module account until the epoch ends
	return nil
}
