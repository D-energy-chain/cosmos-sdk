package distribution_test

import (
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtestutil "github.com/cosmos/cosmos-sdk/x/distribution/testutil"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

// Mock EpochKeeper for testing
type mockEpochKeeper struct {
	isEpochEnd bool
}

func (m *mockEpochKeeper) IsEpochEnd(ctx sdk.Context, identifier string) bool {
	return m.isEpochEnd
}

var (
	valConsAddr0 = sdk.ConsAddress([]byte("val0"))
	valConsAddr1 = sdk.ConsAddress([]byte("val1"))
	distrAcc     = authtypes.NewEmptyModuleAccount(disttypes.ModuleName)
)

func TestEpochBasedDistribution(t *testing.T) {
	ctrl := gomock.NewController(t)
	key := storetypes.NewKVStoreKey(disttypes.StoreKey)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(distribution.AppModuleBasic{})
	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now()})

	bankKeeper := distrtestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := distrtestutil.NewMockStakingKeeper(ctrl)
	accountKeeper := distrtestutil.NewMockAccountKeeper(ctrl)
	feeCollectorAcc := authtypes.NewEmptyModuleAccount("fee_collector")

	valCodec := address.NewBech32Codec("cosmosvaloper")

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(distrAcc.GetAddress())
	accountKeeper.EXPECT().GetModuleAccount(gomock.Any(), "fee_collector").Return(feeCollectorAcc).AnyTimes()
	stakingKeeper.EXPECT().ValidatorAddressCodec().Return(valCodec).AnyTimes()

	distrKeeper := keeper.NewKeeper(
		encCfg.Codec,
		storeService,
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		"fee_collector",
		authtypes.NewModuleAddress("gov").String(),
	)

	// Set up parameters
	err := distrKeeper.Params.Set(ctx, disttypes.DefaultParams())
	require.NoError(t, err)
	require.NoError(t, distrKeeper.FeePool.Set(ctx, disttypes.InitialFeePool()))

	// Create mock epoch keeper
	mockEpoch := &mockEpochKeeper{
		isEpochEnd: false,
	}

	// Set up voting info in context
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx = sdkCtx.WithVoteInfos([]abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsAddr0,
				Power:   100,
			},
		},
		{
			Validator: abci.Validator{
				Address: valConsAddr1,
				Power:   100,
			},
		},
	})

	// Test 1: Not at epoch end - fees should be collected but not distributed
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)))

	// Run BeginBlocker
	err = distribution.BeginBlocker(sdkCtx, distrKeeper, mockEpoch)
	require.NoError(t, err)

	// Test 2: At epoch end - fees should be collected and distributed
	mockEpoch.isEpochEnd = true

	// Now expect the transfer and allocation to happen
	bankKeeper.EXPECT().GetAllBalances(gomock.Any(), feeCollectorAcc.GetAddress()).Return(fees)
	bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", disttypes.ModuleName, fees)

	// Run BeginBlocker at epoch end
	err = distribution.BeginBlocker(sdkCtx, distrKeeper, mockEpoch)
	require.NoError(t, err)

	// Verify community pool received its portion (community tax is 2% by default)
	feePool, err := distrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.False(t, feePool.CommunityPool.IsZero(), "Community pool should have received funds")
}
