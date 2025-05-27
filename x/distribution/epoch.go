package distribution

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/exported"
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

// AllEpochInfos returns all epoch infos
func (k *SimpleEpochKeeper) AllEpochInfos(ctx sdk.Context) []exported.EpochInfo {
	epochs := make([]exported.EpochInfo, len(k.epochIdentifiers))
	for i, identifier := range k.epochIdentifiers {
		epochs[i] = exported.EpochInfo{
			Identifier: identifier,
		}
	}
	return epochs
}

// IsEpochEnd returns true if the current block is the end of an epoch
func (k *SimpleEpochKeeper) IsEpochEnd(ctx sdk.Context, identifier string) bool {
	return k.isEpochEnd[identifier]
}

// SetEpochEnd sets whether the current block is the end of an epoch
func (k *SimpleEpochKeeper) SetEpochEnd(identifier string, isEnd bool) {
	k.isEpochEnd[identifier] = isEnd
}
