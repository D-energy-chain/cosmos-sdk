package distribution

import (
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

// CreateEpochBasedDistributionHelper creates a helper for epoch-based distribution
// that can be used by the parent codebase to integrate with the distribution module.
func CreateEpochBasedDistributionHelper(k keeper.Keeper) *EpochBasedDistributionHelper {
	return &EpochBasedDistributionHelper{
		keeper: k,
	}
}

// EpochBasedDistributionHelper provides helper functions for epoch-based distribution
type EpochBasedDistributionHelper struct {
	keeper keeper.Keeper
}

// ProcessBlock should be called in every block to record the proposer
// This ensures the proposer is properly tracked for reward distribution
func (h *EpochBasedDistributionHelper) ProcessBlock(ctx types.Context) error {
	consAddr := types.ConsAddress(ctx.BlockHeader().ProposerAddress)
	return h.keeper.SetPreviousProposerConsAddr(ctx, consAddr)
}

// ProcessEpochEnd should be called at the end of an epoch to distribute rewards
// This function will distribute all accumulated rewards since the last epoch
func (h *EpochBasedDistributionHelper) ProcessEpochEnd(ctx types.Context) error {
	// determine the total power signing the block
	var previousTotalPower int64
	for _, voteInfo := range ctx.VoteInfos() {
		previousTotalPower += voteInfo.Validator.Power
	}

	// Distribute all accumulated rewards
	return h.keeper.AllocateTokens(ctx, previousTotalPower, ctx.VoteInfos())
}
