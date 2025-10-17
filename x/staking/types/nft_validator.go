package types

import (
	"cosmossdk.io/math"
)

// AddNFTFromDel adds tokens to a validator
func (v Validator) AddNFTFromDel(delegatorAddress, nftContractAddr string, tokenId, amount math.Int) (Validator, math.LegacyDec) {
	// calculate the shares to issue
	var issuedNFTs math.LegacyDec
	if v.DelegatorNftShares.IsNil() || v.DelegatorNftShares.IsZero() {
		// the first delegation to a validator sets the exchange rate to one
		issuedNFTs = math.LegacyNewDecFromInt(amount)
	} else {
		shares, err := v.SharesFromNFTs(amount)
		if err != nil {
			panic(err)
		}

		issuedNFTs = shares
	}

	// Ensure fields are not nil before operations
	if v.TotalNftDelegation.IsNil() {
		v.TotalNftDelegation = math.ZeroInt()
	}
	if v.DelegatorNftShares.IsNil() {
		v.DelegatorNftShares = math.LegacyZeroDec()
	}

	v.TotalNftDelegation = v.TotalNftDelegation.Add(amount)
	v.DelegatorNftShares = v.DelegatorNftShares.Add(issuedNFTs)

	// Create a new NFTDelegation and append it to the NftDelegations array
	nftDelegation := &NFTDelegation{
		DelegatorAddress:   delegatorAddress, // Using operator address as delegator address
		ValidatorAddress:   v.OperatorAddress,
		NftContractAddress: nftContractAddr,
		TokenId:            tokenId.Uint64(), // Convert math.Int to uint64
		Shares:             issuedNFTs,
	}

	v.NftDelegations = append(v.NftDelegations, nftDelegation)

	return v, issuedNFTs
}

func (v Validator) GetNFTDelegatorShares() math.LegacyDec {
	if v.DelegatorNftShares.IsNil() {
		return math.LegacyZeroDec()
	}
	return v.DelegatorNftShares
}

// calculate the token worth of provided shares
func (v Validator) NFTFromShares(shares math.LegacyDec) math.LegacyDec {
	if v.TotalNftDelegation.IsNil() || v.DelegatorNftShares.IsNil() || v.DelegatorNftShares.IsZero() {
		return math.LegacyZeroDec()
	}
	return (shares.MulInt(v.TotalNftDelegation)).Quo(v.DelegatorNftShares)
}

// calculate the token worth of provided shares, truncated
func (v Validator) NFTFromSharesTruncated(shares math.LegacyDec) math.LegacyDec {
	if v.TotalNftDelegation.IsNil() || v.DelegatorNftShares.IsNil() || v.DelegatorNftShares.IsZero() {
		return math.LegacyZeroDec()
	}
	return (shares.MulInt(v.TotalNftDelegation)).QuoTruncate(v.DelegatorNftShares)
}

// TokensFromSharesRoundUp returns the token worth of provided shares, rounded
// up.
func (v Validator) NFTFromSharesRoundUp(shares math.LegacyDec) math.LegacyDec {
	if v.TotalNftDelegation.IsNil() || v.DelegatorNftShares.IsNil() || v.DelegatorNftShares.IsZero() {
		return math.LegacyZeroDec()
	}
	return (shares.MulInt(v.TotalNftDelegation)).QuoRoundUp(v.DelegatorNftShares)
}

// SharesFromTokens returns the shares of a delegation given a bond amount. It
// returns an error if the validator has no tokens.
func (v Validator) SharesFromNFTs(amt math.Int) (math.LegacyDec, error) {
	if v.TotalNftDelegation.IsNil() || v.TotalNftDelegation.IsZero() {
		return math.LegacyZeroDec(), ErrInsufficientShares
	}

	return v.GetNFTDelegatorShares().MulInt(amt).QuoInt(v.GetTotalNFTs()), nil
}

// SharesFromTokensTruncated returns the truncated shares of a delegation given
// a bond amount. It returns an error if the validator has no tokens.
func (v Validator) SharesFromNFTsTruncated(amt math.Int) (math.LegacyDec, error) {
	if v.TotalNftDelegation.IsNil() || v.TotalNftDelegation.IsZero() {
		return math.LegacyZeroDec(), ErrInsufficientShares
	}

	return v.GetNFTDelegatorShares().MulInt(amt).QuoTruncate(math.LegacyNewDecFromInt(v.GetTotalNFTs())), nil
}

// RemoveDelShares removes delegator shares from a validator.
// NOTE: because token fractions are left in the valiadator,
//
//	the exchange rate of future shares of this validator can increase.
func (v Validator) RemoveDelNFTShares(delShares math.LegacyDec) (Validator, math.Int) {
	// Ensure fields are not nil before operations
	if v.DelegatorNftShares.IsNil() {
		v.DelegatorNftShares = math.LegacyZeroDec()
	}
	if v.TotalNftDelegation.IsNil() {
		v.TotalNftDelegation = math.ZeroInt()
	}

	remainingShares := v.DelegatorNftShares.Sub(delShares)

	var issuedTokens math.Int
	if remainingShares.IsZero() {
		// last delegation share gets any trimmings
		issuedTokens = v.TotalNftDelegation
		v.TotalNftDelegation = math.ZeroInt()
	} else {
		// leave excess tokens in the validator
		// however fully use all the delegator shares
		issuedTokens = v.NFTFromShares(delShares).TruncateInt()
		v.TotalNftDelegation = v.TotalNftDelegation.Sub(issuedTokens)

		if v.TotalNftDelegation.IsNegative() {
			panic("attempting to remove more NFTs than available in validator")
		}
	}

	v.DelegatorNftShares = remainingShares

	return v, issuedTokens
}


