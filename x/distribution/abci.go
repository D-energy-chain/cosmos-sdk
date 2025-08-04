package distribution

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/exported"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

// BeginBlocker sets the proposer for determining distribution during endblock
// and accumulates rewards for the current block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper, ek exported.EpochKeeper) error {

	return nil
}
