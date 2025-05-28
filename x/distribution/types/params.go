package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// DefaultParams returns default distribution parameters
func DefaultParams() Params {
	return Params{
		CommunityTax:        math.LegacyNewDecWithPrec(20, 2), // 20%
		BaseProposerReward:  math.LegacyZeroDec(),             // deprecated
		BonusProposerReward: math.LegacyZeroDec(),             // deprecated
		WithdrawAddrEnabled: true,
		NftStakingRatio:     math.LegacyNewDecWithPrec(75, 2), // 75% of staking rewards (56.25% of total)
		NativeStakingRatio:  math.LegacyNewDecWithPrec(25, 2), // 25% of staking rewards (18.75% of total)
		SeparatePoolRatio:   math.LegacyNewDecWithPrec(5, 2),  // 5% to separate pool
	}
}

// ValidateBasic performs basic validation on distribution parameters.
func (p Params) ValidateBasic() error {
	if err := validateCommunityTax(p.CommunityTax); err != nil {
		return err
	}
	if err := validateStakingRatio(p.NftStakingRatio, "nft_staking_ratio"); err != nil {
		return err
	}
	if err := validateStakingRatio(p.NativeStakingRatio, "native_staking_ratio"); err != nil {
		return err
	}

	// Validate that the sum of staking ratios equals 100% (1.0) of the staking rewards
	// Note: The staking rewards are the portion after community tax is deducted
	totalStakingRatio := p.NftStakingRatio.Add(p.NativeStakingRatio)
	expectedTotalStakingRatio := math.LegacyOneDec() // 100% of staking rewards
	if !totalStakingRatio.Equal(expectedTotalStakingRatio) {
		return fmt.Errorf("nft_staking_ratio + native_staking_ratio must equal 100%% (1.0) of staking rewards, got: %s", totalStakingRatio)
	}

	return nil
}

func validateCommunityTax(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("community tax must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("community tax must be positive: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("community tax too large: %s", v)
	}

	return nil
}

func validateWithdrawAddrEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateStakingRatio(i interface{}, paramName string) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type for %s: %T", paramName, i)
	}

	if v.IsNil() {
		return fmt.Errorf("%s must be not nil", paramName)
	}
	if v.IsNegative() {
		return fmt.Errorf("%s must be positive: %s", paramName, v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("%s too large: %s", paramName, v)
	}

	return nil
}
