package types

import sdkmath "cosmossdk.io/math"

// create a new DelegatorStartingInfo
func NewDelegatorStartingInfo(previousPeriod uint64, stake sdkmath.LegacyDec, height uint64) DelegatorStartingInfo {
	return DelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		NftStake:       sdkmath.LegacyZeroDec(),
		Height:         height,
	}
}

// create a new DelegatorStartingInfo with both native and NFT stakes
func NewDelegatorStartingInfoWithNFT(previousPeriod uint64, stake, nftStake sdkmath.LegacyDec, height uint64) DelegatorStartingInfo {
	return DelegatorStartingInfo{
		PreviousPeriod: previousPeriod,
		Stake:          stake,
		NftStake:       nftStake,
		Height:         height,
	}
}
