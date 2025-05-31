package keeper_test

import (
	"github.com/golang/mock/gomock"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	// Use additional keys for the test
	TestPK       = PKS[1] // Use second key from the existing PKS slice
	TestAddr     = sdk.AccAddress(TestPK.Address())
	TestValAddr  = sdk.ValAddress(TestAddr)
	TestPK2      = PKS[2] // Use third key
	TestAddr2    = sdk.AccAddress(TestPK2.Address())
	TestValAddr2 = sdk.ValAddress(TestAddr2)
)

func (s *KeeperTestSuite) TestCreateValidatorWhitelistEnforcement() {
	ctx, require := s.ctx, s.Require()

	// Set up mock expectations for our test addresses
	s.bankKeeper.EXPECT().DelegateCoinsFromAccountToModule(gomock.Any(), TestAddr, stakingtypes.NotBondedPoolName, gomock.Any()).AnyTimes()
	s.bankKeeper.EXPECT().DelegateCoinsFromAccountToModule(gomock.Any(), TestAddr2, stakingtypes.NotBondedPoolName, gomock.Any()).AnyTimes()

	msgServer := keeper.NewMsgServerImpl(s.stakingKeeper)

	// Create valid test message
	createValidatorMsg := func(delegatorAddr, validatorAddr string) *stakingtypes.MsgCreateValidator {
		pk1 := ed25519.GenPrivKey().PubKey()
		pubkey, err := codectypes.NewAnyWithValue(pk1)
		require.NoError(err)

		return &stakingtypes.MsgCreateValidator{
			Description: stakingtypes.Description{
				Moniker: "TestValidator",
			},
			Commission: stakingtypes.CommissionRates{
				Rate:          math.LegacyNewDecWithPrec(5, 1),
				MaxRate:       math.LegacyNewDecWithPrec(5, 1),
				MaxChangeRate: math.LegacyNewDec(0),
			},
			MinSelfDelegation: math.NewInt(1),
			DelegatorAddress:  delegatorAddr,
			ValidatorAddress:  validatorAddr,
			Pubkey:            pubkey,
			Value:             sdk.NewInt64Coin("stake", 10000),
		}
	}

	// Test 1: Empty whitelist allows any validator
	err := s.stakingKeeper.SetParams(ctx, stakingtypes.Params{
		UnbondingTime:     stakingtypes.DefaultUnbondingTime,
		MaxValidators:     stakingtypes.DefaultMaxValidators,
		MaxEntries:        stakingtypes.DefaultMaxEntries,
		HistoricalEntries: stakingtypes.DefaultHistoricalEntries,
		BondDenom:         "stake",
		MinCommissionRate: math.LegacyZeroDec(),
		AllowedValidators: []string{}, // Empty whitelist
	})
	require.NoError(err)

	msg := createValidatorMsg(TestAddr.String(), TestValAddr.String())
	_, err = msgServer.CreateValidator(ctx, msg)
	require.NoError(err, "Empty whitelist should allow any validator")

	// Test 2: Validator not in whitelist should fail
	err = s.stakingKeeper.SetParams(ctx, stakingtypes.Params{
		UnbondingTime:     stakingtypes.DefaultUnbondingTime,
		MaxValidators:     stakingtypes.DefaultMaxValidators,
		MaxEntries:        stakingtypes.DefaultMaxEntries,
		HistoricalEntries: stakingtypes.DefaultHistoricalEntries,
		BondDenom:         "stake",
		MinCommissionRate: math.LegacyZeroDec(),
		AllowedValidators: []string{TestValAddr.String()}, // Only allow first validator
	})
	require.NoError(err)

	msg = createValidatorMsg(TestAddr2.String(), TestValAddr2.String())
	_, err = msgServer.CreateValidator(ctx, msg)
	require.Error(err, "Validator not in whitelist should fail")
	require.Contains(err.Error(), "not in the allowed validators list", "Error should mention validator not allowed")

	// Test 3: Validator in whitelist should succeed (add second validator to whitelist)
	err = s.stakingKeeper.SetParams(ctx, stakingtypes.Params{
		UnbondingTime:     stakingtypes.DefaultUnbondingTime,
		MaxValidators:     stakingtypes.DefaultMaxValidators,
		MaxEntries:        stakingtypes.DefaultMaxEntries,
		HistoricalEntries: stakingtypes.DefaultHistoricalEntries,
		BondDenom:         "stake",
		MinCommissionRate: math.LegacyZeroDec(),
		AllowedValidators: []string{TestValAddr.String(), TestValAddr2.String()}, // Allow both validators
	})
	require.NoError(err)

	msg = createValidatorMsg(TestAddr2.String(), TestValAddr2.String())
	_, err = msgServer.CreateValidator(ctx, msg)
	require.NoError(err, "Validator in whitelist should succeed")
}
