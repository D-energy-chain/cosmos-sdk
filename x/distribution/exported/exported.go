package exported

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

type (
	ParamSet = paramtypes.ParamSet

	// Subspace defines an interface that implements the legacy x/params Subspace
	// type.
	//
	// NOTE: This is used solely for migration of x/params managed parameters.
	Subspace interface {
		GetParamSet(ctx sdk.Context, ps ParamSet)
	}
)

// EpochKeeper defines the expected epoch keeper interface that the parent codebase
// should implement to enable epoch-based reward distribution in the distribution module.
type EpochKeeper interface {
	// IsEpochEnd returns true if the current block is the end of an epoch
	IsEpochEnd(ctx sdk.Context, identifier string) bool
}
