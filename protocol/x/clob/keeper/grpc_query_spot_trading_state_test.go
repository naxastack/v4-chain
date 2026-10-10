package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSpotTradingStateQueries(t *testing.T) {
	memClob := memclob.NewMemClobPriceTimePriority(false)
	ks := keepertest.NewClobKeepersTestContext(t, memClob, &mocks.BankKeeper{}, &mocks.IndexerEventManager{})
	owner := constants.AliceAccAddress.String()

	ks.ClobKeeper.SetClobPair(ks.Ctx, types.ClobPair{
		Id: 8,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId:  1,
			QuoteAssetId: 0,
		}},
		Status: types.ClobPair_STATUS_ACTIVE,
	})
	ks.ClobKeeper.SetSpotPairLifecycleState(ks.Ctx, 8, types.SpotPairLifecycleState{
		PauseSource:            types.SpotPauseSourceManual,
		FinalSettlementStarted: true,
	})
	cursor := types.SpotDelistCursor{ClobPairId: 8}
	ks.ClobKeeper.SetSpotDelistCursor(ks.Ctx, cursor)
	orderID := types.OrderId{
		SubaccountId: satypes.SubaccountId{Owner: owner, Number: 1},
		ClientId:     4,
		ClobPairId:   8,
		OrderFlags:   types.OrderIdFlags_LongTerm,
	}
	ks.ClobKeeper.SetOrderFillAmount(ks.Ctx, orderID, 15, 20)

	pairRes, err := ks.ClobKeeper.ClobPair(ks.Ctx, &types.QueryGetClobPairRequest{Id: 8})
	require.NoError(t, err)
	require.Equal(t, uint32(1), pairRes.ClobPair.GetSpotClobMetadata().BaseAssetId)
	require.Equal(t, uint32(0), pairRes.ClobPair.GetSpotClobMetadata().QuoteAssetId)

	lifecycle, err := ks.ClobKeeper.SpotPairLifecycle(ks.Ctx, &types.QuerySpotPairLifecycleRequest{ClobPairId: 8})
	require.NoError(t, err)
	require.Equal(t, types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL, lifecycle.PauseSource)
	require.True(t, lifecycle.FinalSettlementStarted)

	_, err = ks.ClobKeeper.SpotPairLifecycle(ks.Ctx, &types.QuerySpotPairLifecycleRequest{ClobPairId: 99})
	require.Equal(t, codes.NotFound, status.Code(err))

	delist, err := ks.ClobKeeper.SpotDelistCursor(ks.Ctx, &types.QuerySpotDelistCursorRequest{ClobPairId: 8})
	require.NoError(t, err)
	require.Equal(t, cursor, delist.Cursor)
	require.Equal(t, types.OrderId{}, delist.Cursor.NextOrderId)

	fill, err := ks.ClobKeeper.OrderFillAmount(ks.Ctx, &types.QueryOrderFillAmountRequest{
		Owner:      owner,
		Number:     1,
		ClientId:   4,
		ClobPairId: 8,
		OrderFlags: types.OrderIdFlags_LongTerm,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(15), fill.FillAmount)

	missing, err := ks.ClobKeeper.OrderFillAmount(ks.Ctx, &types.QueryOrderFillAmountRequest{
		Owner:      owner,
		Number:     1,
		ClientId:   5,
		ClobPairId: 8,
		OrderFlags: types.OrderIdFlags_ShortTerm,
	})
	require.NoError(t, err)
	require.Zero(t, missing.FillAmount)

	ks.ClobKeeper.SetLongTermOrderPlacement(ks.Ctx, types.Order{
		OrderId:          orderID,
		Side:             types.Order_SIDE_BUY,
		Quantums:         10,
		Subticks:         20,
		GoodTilOneof:     &types.Order_GoodTilBlockTime{GoodTilBlockTime: 100},
		MaxTradingFeePpm: 1_000,
	}, 1)
	stateful, err := ks.ClobKeeper.StatefulOrder(ks.Ctx, &types.QueryStatefulOrderRequest{OrderId: orderID})
	require.NoError(t, err)
	require.Equal(t, uint32(1_000), stateful.OrderPlacement.Order.MaxTradingFeePpm)
	require.Equal(t, uint64(15), stateful.FillAmount)
}
