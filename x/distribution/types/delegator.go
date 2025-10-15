package types

import sdkmath "cosmossdk.io/math"

// create a new DelegatorStartingInfo
func NewDelegatorStartingInfo(previousPeriod uint64, stake sdkmath.LegacyDec, height uint64) DelegatorStartingInfo {
	return DelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		Height:         height,
	}
}

// create a new NFTDelegatorStartingInfo
func NewNFTDelegatorStartingInfo(previousPeriod uint64, stake sdkmath.LegacyDec, height uint64) NFTDelegatorStartingInfo {
	return NFTDelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		NftStake:       stake,
		Height:         height,
	}
}
