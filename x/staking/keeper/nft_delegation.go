package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// SetDelegation sets a delegation.
func (k Keeper) GetNFTDelegationShares(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (types.NFTDelegationShares, error) {
	return k.NFTDelShares.Get(ctx, collections.Join(delAddr.Bytes(), valAddr.Bytes()))
}

// SetDelegation sets a delegation.
func (k Keeper) SetNFTDelegationShares(ctx context.Context, delegation types.NFTDelegationShares) error {
	delegatorAddress, err := k.authKeeper.AddressCodec().StringToBytes(delegation.DelegatorAddress)
	if err != nil {
		return err
	}

	valBz, err := k.ValidatorAddressCodec().StringToBytes(delegation.GetValidatorAddr())
	if err != nil {
		return err
	}

	err = k.NFTDelShares.Set(ctx, collections.Join(delegatorAddress, valBz), delegation)
	return err
}

// UpdateNFTDelegationShares updates or creates NFT delegation shares for a delegator/validator pair.
// If add is true, sharesDelta will be added; otherwise it will be subtracted. If no existing
// record is found, a new one is created with zero shares prior to applying the delta.
func (k Keeper) UpdateNFTDelegationShares(
	ctx context.Context,
	delAddr sdk.AccAddress,
	valAddr sdk.ValAddress,
	sharesDelta sdkmath.LegacyDec,
	add bool,
) error {
	// Try to load existing record
	existing, err := k.GetNFTDelegationShares(ctx, delAddr, valAddr)
	switch {
	case err == nil:
		// ok
	case errors.IsOf(err, collections.ErrNotFound):
		// Create a new record with zero shares
		delStr, derr := k.authKeeper.AddressCodec().BytesToString(delAddr)
		if derr != nil {
			return derr
		}
		valStr, verr := k.ValidatorAddressCodec().BytesToString(valAddr)
		if verr != nil {
			return verr
		}
		existing = types.NewNFTDelegationShares(delStr, valStr, sdkmath.LegacyZeroDec())
	default:
		return err
	}

	// Apply delta
	if add {
		existing.Shares = existing.Shares.Add(sharesDelta)
	} else {
		existing.Shares = existing.Shares.Sub(sharesDelta)
		if existing.Shares.IsNegative() {
			existing.Shares = sdkmath.LegacyZeroDec()
		}
	}

	// Persist
	return k.SetNFTDelegationShares(ctx, existing)
}
