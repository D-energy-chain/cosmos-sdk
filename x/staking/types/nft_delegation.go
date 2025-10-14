package types

import (
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
)

// Implements NFTDelegation interface
var _ NFTDelegationI = NFTDelegationShares{}

// NewDelegation creates a new delegation object
//
//nolint:interfacer
func NewNFTDelegationShares(delegatorAddr, validatorAddr string, bondAmt math.LegacyDec) NFTDelegationShares {
	return NFTDelegationShares{
		DelegatorAddress: delegatorAddr,
		ValidatorAddress: validatorAddr,
		Shares:           bondAmt,
	}
}

// MustMarshalDelegation returns the delegation bytes. Panics if fails
func MustMarshalNFTDelegationShares(cdc codec.BinaryCodec, delegation NFTDelegationShares) []byte {
	return cdc.MustMarshal(&delegation)
}

// MustUnmarshalDelegation return the unmarshaled delegation from bytes.
// Panics if fails.
func MustUnmarshalNFTDelegationShares(cdc codec.BinaryCodec, value []byte) NFTDelegationShares {
	delegation, err := UnmarshalNFTDelegationShares(cdc, value)
	if err != nil {
		panic(err)
	}

	return delegation
}

// return the delegation
func UnmarshalNFTDelegationShares(cdc codec.BinaryCodec, value []byte) (delegation NFTDelegationShares, err error) {
	err = cdc.Unmarshal(value, &delegation)
	return delegation, err
}

func (d NFTDelegationShares) GetDelegatorAddr() string {
	return d.DelegatorAddress
}

func (d NFTDelegationShares) GetValidatorAddr() string {
	return d.ValidatorAddress
}

func (d NFTDelegationShares) GetNFTShares() math.LegacyDec { return d.Shares }
