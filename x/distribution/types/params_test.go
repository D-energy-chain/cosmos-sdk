package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func TestParams_ValidateBasic(t *testing.T) {
	toDec := sdkmath.LegacyMustNewDecFromStr

	type fields struct {
		BaseProposerReward  sdkmath.LegacyDec
		BonusProposerReward sdkmath.LegacyDec
		WithdrawAddrEnabled bool
		NftStakingRatio     sdkmath.LegacyDec
		NativeStakingRatio  sdkmath.LegacyDec
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"success", fields{toDec("0"), toDec("0"), false, toDec("0.75"), toDec("0.25")}, false},
		{"negative base proposer reward (must not matter)", fields{toDec("-0.1"), toDec("0"), false, toDec("0.75"), toDec("0.25")}, false},
		{"negative bonus proposer reward (must not matter)", fields{toDec("0"), toDec("-0.1"), false, toDec("0.75"), toDec("0.25")}, false},
		{"negative nft staking ratio", fields{toDec("0"), toDec("0"), false, toDec("-0.1"), toDec("0.25")}, true},
		{"negative native staking ratio", fields{toDec("0"), toDec("0"), false, toDec("0.75"), toDec("-0.1")}, true},
		{"staking ratios don't sum to 1", fields{toDec("0"), toDec("0"), false, toDec("0.5"), toDec("0.3")}, true},
		{"nft staking ratio greater than 1", fields{toDec("0"), toDec("0"), false, toDec("1.1"), toDec("0.25")}, true},
		{"native staking ratio greater than 1", fields{toDec("0"), toDec("0"), false, toDec("0.75"), toDec("1.1")}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := types.Params{
				WithdrawAddrEnabled: tt.fields.WithdrawAddrEnabled,
				NftStakingRatio:     tt.fields.NftStakingRatio,
				NativeStakingRatio:  tt.fields.NativeStakingRatio,
			}
			if err := p.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().ValidateBasic())
}
