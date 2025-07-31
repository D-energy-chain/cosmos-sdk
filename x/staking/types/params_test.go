package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func TestParamsEqual(t *testing.T) {
	p1 := types.DefaultParams()
	p2 := types.DefaultParams()

	ok := p1.Equal(p2)
	require.True(t, ok)

	p2.UnbondingTime = 60 * 60 * 24 * 2
	p2.BondDenom = "soup"

	ok = p1.Equal(p2)
	require.False(t, ok)
}

func TestValidateParams(t *testing.T) {
	params := types.DefaultParams()

	// default params have no error
	require.NoError(t, params.Validate())

	// validate mincommission
	params.MinCommissionRate = math.LegacyNewDec(-1)
	require.Error(t, params.Validate())

	params.MinCommissionRate = math.LegacyNewDec(2)
	require.Error(t, params.Validate())
}

func TestParamsAllowedValidatorsNilHandling(t *testing.T) {
	// Test that default params have properly initialized AllowedValidators
	params := types.DefaultParams()
	require.NotNil(t, params.AllowedValidators, "AllowedValidators should not be nil")
	require.Empty(t, params.AllowedValidators, "AllowedValidators should be empty slice")
	
	// Test IsValidatorAllowed works correctly with empty slice
	require.True(t, params.IsValidatorAllowed("any-address"), "any validator should be allowed when list is empty")
	
	// Test with specific validators
	params.AllowedValidators = []string{"validator1", "validator2"}
	require.True(t, params.IsValidatorAllowed("validator1"), "validator1 should be allowed")
	require.False(t, params.IsValidatorAllowed("validator3"), "validator3 should not be allowed")
}
