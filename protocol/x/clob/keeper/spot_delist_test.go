package keeper_test

import (
	"testing"
	"time"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	blocktimetypes "github.com/dydxprotocol/v4-chain/protocol/x/blocktime/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdvanceSpotDelistOrdersRotatesOneOrderPerPair(t *testing.T) {
	ks, id := newSpotDelistKeeper(t)
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 1))
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 2))
	require.NoError(t, placeSpotDelistOrder(ks, id, 9, 3))
	require.NoError(t, placeSpotDelistOrder(ks, id, 9, 4))
	enterSpotFinalSettlement(t, ks, 8)
	enterSpotFinalSettlement(t, ks, 9)
	setDelistLimit(t, ks, 1)

	removed := ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx)
	require.Len(t, removed, 1)
	require.Equal(t, uint32(8), removed[0].ClobPairId)
	require.Equal(t, 1, countSpotReservations(ks, 8))
	require.Equal(t, 2, countSpotReservations(ks, 9))

	removed = ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx)
	require.Len(t, removed, 1)
	require.Equal(t, uint32(9), removed[0].ClobPairId)
	require.Equal(t, 1, countSpotReservations(ks, 8))
	require.Equal(t, 1, countSpotReservations(ks, 9))
}

func TestAdvanceSpotDelistOrdersUsesSharedLimitAcrossPairs(t *testing.T) {
	ks, id := newSpotDelistKeeper(t)
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 1))
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 2))
	require.NoError(t, placeSpotDelistOrder(ks, id, 9, 3))
	require.NoError(t, placeSpotDelistOrder(ks, id, 9, 4))
	enterSpotFinalSettlement(t, ks, 8)
	enterSpotFinalSettlement(t, ks, 9)
	setDelistLimit(t, ks, 2)

	removed := ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx)
	require.Len(t, removed, 2)
	require.Equal(t, uint32(8), removed[0].ClobPairId)
	require.Equal(t, uint32(9), removed[1].ClobPairId)
	require.Equal(t, 1, countSpotReservations(ks, 8))
	require.Equal(t, 1, countSpotReservations(ks, 9))
}

func TestAdvanceSpotDelistOrdersSkipsUserCancelledOrder(t *testing.T) {
	ks, id := newSpotDelistKeeper(t)
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 1))
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 2))
	ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false)
	require.NoError(t, ks.ClobKeeper.CancelStatefulOrder(ctx, types.NewMsgCancelOrderStateful(
		types.OrderId{SubaccountId: id, ClientId: 1, OrderFlags: types.OrderIdFlags_LongTerm, ClobPairId: 8},
		100,
	)))
	enterSpotFinalSettlement(t, ks, 8)
	setDelistLimit(t, ks, 100)

	removed := ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx)
	require.Len(t, removed, 1)
	require.Equal(t, uint32(2), removed[0].ClientId)
	require.Zero(t, countSpotReservations(ks, 8))
	stored := ks.SubaccountsKeeper.GetSubaccount(ks.Ctx, id)
	require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	cursor, found := ks.ClobKeeper.GetSpotDelistCursor(ks.Ctx, 8)
	require.True(t, found)
	require.True(t, cursor.Completed)

	require.Empty(t, ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx))
	require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
}

func TestAdvanceSpotDelistOrdersKeepsCursorWhenReleaseFails(t *testing.T) {
	ks, id := newSpotDelistKeeper(t)
	require.NoError(t, placeSpotDelistOrder(ks, id, 8, 1))
	enterSpotFinalSettlement(t, ks, 8)
	reservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ks.Ctx, types.OrderId{
		SubaccountId: id, ClientId: 1, OrderFlags: types.OrderIdFlags_LongTerm, ClobPairId: 8,
	})
	require.True(t, found)
	reservation.ReservedQuantums = dtypes.NewInt(-1)
	ks.ClobKeeper.SetStatefulSpotReservation(ks.Ctx, reservation)
	before, found := ks.ClobKeeper.GetSpotDelistCursor(ks.Ctx, 8)
	require.True(t, found)

	require.Empty(t, ks.ClobKeeper.AdvanceSpotDelistOrders(ks.Ctx))
	after, found := ks.ClobKeeper.GetSpotDelistCursor(ks.Ctx, 8)
	require.True(t, found)
	require.Equal(t, before, after)
	_, found = ks.ClobKeeper.GetStatefulSpotReservation(ks.Ctx, reservation.OrderId)
	require.True(t, found)
}

func newSpotDelistKeeper(t *testing.T) (keepertest.ClobKeepersTestContext, satypes.SubaccountId) {
	t.Helper()
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	indexer.On("AddBlockEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	configureSpotActivationParams(t, ks)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	for _, pairId := range []uint32{8, 9} {
		pair := spotReservationTestPair()
		pair.Id = pairId
		createActiveSpotClobPair(t, ks, pair)
	}
	id := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
	ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
		Id:          &id,
		AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId: assettypes.AssetUsdc.Id, Quantums: dtypes.NewInt(1_000),
			StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	ks.BlockTimeKeeper.SetPreviousBlockInfo(ks.Ctx, &blocktimetypes.BlockInfo{Timestamp: time.Unix(1, 0)})
	ks.Ctx = ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
	return ks, id
}

func placeSpotDelistOrder(ks keepertest.ClobKeepersTestContext, id satypes.SubaccountId, pairId uint32, clientId uint32) error {
	order := spotReservationTestOrder(id, clientId)
	order.OrderId.ClobPairId = pairId
	return ks.ClobKeeper.PlaceStatefulOrder(ks.Ctx, &types.MsgPlaceOrder{Order: order}, true)
}

func enterSpotFinalSettlement(t *testing.T, ks keepertest.ClobKeepersTestContext, pairId uint32) {
	t.Helper()
	require.NoError(t, ks.ClobKeeper.AdvanceSpotAssetLifecycle(ks.Ctx, types.ClobPairId(pairId), types.ClobPair_STATUS_FINAL_SETTLEMENT))
}

func setDelistLimit(t *testing.T, ks keepertest.ClobKeepersTestContext, limit uint32) {
	t.Helper()
	params := types.DefaultSpotResourceParams()
	params.MaxSpotDelistOrdersPerBlock = limit
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, params))
}

func countSpotReservations(ks keepertest.ClobKeepersTestContext, pairId uint32) int {
	count := 0
	for _, reservation := range ks.ClobKeeper.GetAllStatefulSpotReservations(ks.Ctx) {
		if reservation.OrderId.ClobPairId == pairId {
			count++
		}
	}
	return count
}
