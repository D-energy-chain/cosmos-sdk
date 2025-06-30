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

func TestDistributionHelper(t *testing.T) {
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

	accountKeeper.EXPECT().GetModuleAddress("distribution").Return(authtypes.NewEmptyModuleAccount(disttypes.ModuleName).GetAddress())
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

	// Create distribution helper
	helper := distribution.CreateEpochBasedDistributionHelper(distrKeeper)

	// Test ProcessBlock
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	err = helper.ProcessBlock(sdkCtx)
	require.NoError(t, err)

	// Test SimpleEpochKeeper
	simpleKeeper := distribution.NewSimpleEpochKeeper([]string{"day", "week"})

	// Check initial state
	require.False(t, simpleKeeper.IsEpochEnd(sdkCtx, "day"))
	require.False(t, simpleKeeper.IsEpochEnd(sdkCtx, "week"))

	// Set epoch end
	simpleKeeper.SetEpochEnd("day", true)
	require.True(t, simpleKeeper.IsEpochEnd(sdkCtx, "day"))
	require.False(t, simpleKeeper.IsEpochEnd(sdkCtx, "week"))

	// Test ProcessEpochEnd
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1000)))
	bankKeeper.EXPECT().GetAllBalances(gomock.Any(), feeCollectorAcc.GetAddress()).Return(fees)
	bankKeeper.EXPECT().SendCoinsFromModuleToModule(gomock.Any(), "fee_collector", disttypes.ModuleName, fees)

	// Set up voting info in context
	sdkCtx = sdkCtx.WithVoteInfos([]abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: []byte("val1"),
				Power:   100,
			},
		},
		{
			Validator: abci.Validator{
				Address: []byte("val2"),
				Power:   100,
			},
		},
	})

	err = helper.ProcessEpochEnd(sdkCtx)
	require.NoError(t, err)
}
