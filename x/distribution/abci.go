package distribution

import (
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

	// determine the total power signing the block
	var previousTotalPower int64
	for _, voteInfo := range ctx.VoteInfos() {
		previousTotalPower += voteInfo.Validator.Power
	}

	// record the proposer for when we payout on the next block
	if err := SetProposerFromBlockHeader(ctx, k); err != nil {
		return err
	}

	// TODO fix the hardcoded epoch identifier later
	// Check if the epoch has ended
	if ek.IsEpochEnd(ctx, "inflation") {
		// At epoch end, distribute all accumulated rewards
		if err := k.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos()); err != nil {
			return err
		}
		return nil
	}

	// If not at epoch end, just collect fees but don't distribute them
	// The fees will remain in the fee collector module account until the epoch ends
	return nil
}

