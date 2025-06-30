package keeper_test

import (
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (s *KeeperTestSuite) TestValidatorWhitelist() {
	require := s.Require()

	// Test 1: Empty whitelist allows all validators
	params := types.DefaultParams()
	require.Empty(params.AllowedValidators)

	// Any validator address should be allowed when whitelist is empty
	testAddr := "cosmosvaloper1abcdefg"
	require.True(params.IsValidatorAllowed(testAddr))

	// Test 2: Non-empty whitelist restricts validators
	allowedAddrs := []string{
		"cosmosvaloper1allowed1",
		"cosmosvaloper1allowed2",
	}
	params.AllowedValidators = allowedAddrs

	// Allowed validators should pass
	require.True(params.IsValidatorAllowed("cosmosvaloper1allowed1"))
	require.True(params.IsValidatorAllowed("cosmosvaloper1allowed2"))

	// Non-allowed validators should fail
	require.False(params.IsValidatorAllowed("cosmosvaloper1notallowed"))
	require.False(params.IsValidatorAllowed("cosmosvaloper1random"))

	// Test 3: Validation of allowed validators
	err := s.stakingKeeper.SetParams(s.ctx, params)
	require.NoError(err)

	retrievedParams, err := s.stakingKeeper.GetParams(s.ctx)
	require.NoError(err)
	require.Equal(allowedAddrs, retrievedParams.AllowedValidators)
}

func (s *KeeperTestSuite) TestCreateValidatorWithWhitelist() {
	require := s.Require()

	// Set up a whitelist with specific validators
	params := types.DefaultParams()
	allowedAddr := "cosmosvaloper1test123"
	params.AllowedValidators = []string{allowedAddr}

	err := s.stakingKeeper.SetParams(s.ctx, params)
	require.NoError(err)

	// Test that IsValidatorAllowed works correctly with the stored params
	storedParams, err := s.stakingKeeper.GetParams(s.ctx)
	require.NoError(err)

	require.True(storedParams.IsValidatorAllowed(allowedAddr))
	require.False(storedParams.IsValidatorAllowed("cosmosvaloper1notallowed"))
}

func (s *KeeperTestSuite) TestValidateAllowedValidators() {
	require := s.Require()

	// Test validation function

	// Empty list should be valid
	err := types.DefaultParams().Validate()
	require.NoError(err)

	// Valid addresses should pass
	params := types.DefaultParams()
	params.AllowedValidators = []string{
		"cosmosvaloper1valid1",
		"cosmosvaloper1valid2",
	}
	err = params.Validate()
	require.NoError(err)

	// Empty string should fail validation
	params.AllowedValidators = []string{""}
	err = params.Validate()
	require.Error(err)
	require.Contains(err.Error(), "allowed validator address cannot be blank")

	// Blank string should fail validation
	params.AllowedValidators = []string{"   "}
	err = params.Validate()
	require.Error(err)
	require.Contains(err.Error(), "allowed validator address cannot be blank")
}
