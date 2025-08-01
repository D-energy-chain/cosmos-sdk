package distribution

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SimpleEpochKeeper is a simple implementation of the EpochKeeper interface
type SimpleEpochKeeper struct {
	epochIdentifiers []string
	isEpochEnd       map[string]bool
}

// NewSimpleEpochKeeper creates a new SimpleEpochKeeper
func NewSimpleEpochKeeper(epochIdentifiers []string) *SimpleEpochKeeper {
	return &SimpleEpochKeeper{
		epochIdentifiers: epochIdentifiers,
		isEpochEnd:       make(map[string]bool),
	}
}

// IsEpochEnd returns true if the current block is the end of an epoch
func (k *SimpleEpochKeeper) IsEpochEnd(ctx sdk.Context, identifier string) bool {
	isEnd := k.isEpochEnd[identifier]
	ctx.Logger().Debug("SimpleEpochKeeper.IsEpochEnd called", 
		"identifier", identifier, 
		"is_epoch_end", isEnd, 
		"height", ctx.BlockHeight())
	return isEnd
}

// SetEpochEnd sets whether the current block is the end of an epoch
func (k *SimpleEpochKeeper) SetEpochEnd(identifier string, isEnd bool) {
	k.isEpochEnd[identifier] = isEnd
}
