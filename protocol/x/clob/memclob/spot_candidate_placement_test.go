package memclob

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/memclob"
	sdktest "github.com/dydxprotocol/v4-chain/protocol/testutil/sdk"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func newSpotCandidatePlacementFixture(t *testing.T) (*MemClobPriceTimePriority, types.Order, types.ClobPair) {
	t.Helper()
	pair := types.ClobPair{
		Id: 1,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 5, SubticksPerTick: 1, QuantumConversionExponent: 0,
	}
	order := constants.Order_Alice_Num0_Id0_Clob0_Buy5_Price10_GTB15
	order.OrderId.ClobPairId = 1
	order.OrderId.OrderFlags = types.OrderIdFlags_ShortTerm
	order.Quantums = 10
	order.Subticks = 2
	order.MaxTradingFeePpm = 1_000
	order.OrderEpoch = 3
	memClob := NewMemClobPriceTimePriority(false)
	memClob.CreateOrderbook(pair)
	return memClob, order, pair
}

func TestPlaceSpotShortTermOrderReservesAndCancelReleases(t *testing.T) {
	ctx, _, _ := sdktest.NewSdkContextWithMultistore()
	ctx = ctx.WithIsCheckTx(true).WithTxBytes([]byte{1, 2, 3})
	memClob, order, pair := newSpotCandidatePlacementFixture(t)
	fakeKeeper := memclob.NewFakeMemClobKeeper().WithSpotCandidateStateFn(
		func(_ sdk.Context, got types.Order) (types.SpotCandidateState, error) {
			return types.SpotCandidateState{ClobPair: pair, OutgoingAssetId: 0, TotalQuantums: big.NewInt(22), StatefulReservedQuantums: new(big.Int), CurrentOrderEpoch: got.OrderEpoch}, nil
		},
	)
	memClob.SetClobKeeper(fakeKeeper)

	_, status, _, err := memClob.PlaceOrder(ctx, order)
	require.NoError(t, err)
	require.Equal(t, types.Success, status)
	reservation, found := memClob.spotCandidateLedger.Get(order.OrderId)
	require.True(t, found)
	require.Equal(t, big.NewInt(22), reservation.ReservedQuantums)
	require.False(t, reservation.PendingConfirmation)

	_, err = memClob.CancelOrder(ctx, &types.MsgCancelOrder{OrderId: order.OrderId, GoodTilOneof: &types.MsgCancelOrder_GoodTilBlock{GoodTilBlock: order.GetGoodTilBlock()}})
	require.NoError(t, err)
	_, found = memClob.spotCandidateLedger.Get(order.OrderId)
	require.False(t, found)
}

func TestPlaceSpotShortTermOrderBalanceAndEpochFailuresHaveNoSideEffects(t *testing.T) {
	for _, tc := range []struct {
		name         string
		total        int64
		currentEpoch uint64
		wantErr      error
	}{
		{name: "insufficient balance", total: 21, currentEpoch: 3, wantErr: ErrInsufficientSpotBalance},
		{name: "stale epoch", total: 22, currentEpoch: 4, wantErr: types.ErrSpotOrderEpochMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, _ := sdktest.NewSdkContextWithMultistore()
			ctx = ctx.WithIsCheckTx(true).WithTxBytes([]byte{1})
			memClob, order, pair := newSpotCandidatePlacementFixture(t)
			memClob.SetClobKeeper(memclob.NewFakeMemClobKeeper().WithSpotCandidateStateFn(
				func(_ sdk.Context, _ types.Order) (types.SpotCandidateState, error) {
					return types.SpotCandidateState{ClobPair: pair, OutgoingAssetId: 0, TotalQuantums: big.NewInt(tc.total), StatefulReservedQuantums: new(big.Int), CurrentOrderEpoch: tc.currentEpoch}, nil
				},
			))
			_, _, _, err := memClob.PlaceOrder(ctx, order)
			require.ErrorIs(t, err, tc.wantErr)
			_, found := memClob.GetOrder(order.OrderId)
			require.False(t, found)
			require.Zero(t, memClob.spotCandidateLedger.OrderCount())
		})
	}
}

func TestMatchedSpotShortTermOrdersRemainPendingUntilCommit(t *testing.T) {
	ctx, _, _ := sdktest.NewSdkContextWithMultistore()
	ctx = ctx.WithIsCheckTx(true).WithTxBytes([]byte{1, 2, 3})
	memClob, buy, pair := newSpotCandidatePlacementFixture(t)
	sell := buy
	sell.OrderId.ClientId = 2
	sell.OrderId.SubaccountId.Number++
	sell.Side = types.Order_SIDE_SELL
	memClob.SetClobKeeper(memclob.NewFakeMemClobKeeper().WithSpotCandidateStateFn(
		func(_ sdk.Context, order types.Order) (types.SpotCandidateState, error) {
			assetId := uint32(0)
			if !order.IsBuy() {
				assetId = 1
			}
			return types.SpotCandidateState{ClobPair: pair, OutgoingAssetId: assetId, TotalQuantums: big.NewInt(100), StatefulReservedQuantums: new(big.Int), CurrentOrderEpoch: order.OrderEpoch}, nil
		},
	))

	_, status, _, err := memClob.PlaceOrder(ctx, sell)
	require.NoError(t, err)
	require.Equal(t, types.Success, status)
	filled, status, _, err := memClob.PlaceOrder(ctx, buy)
	require.NoError(t, err)
	require.Equal(t, types.Success, status)
	require.Equal(t, buy.GetBaseQuantums(), filled)

	buyReservation, found := memClob.spotCandidateLedger.Get(buy.OrderId)
	require.True(t, found)
	require.True(t, buyReservation.PendingConfirmation)
	sellReservation, found := memClob.spotCandidateLedger.Get(sell.OrderId)
	require.True(t, found)
	require.True(t, sellReservation.PendingConfirmation)

	_, err = memClob.CancelOrder(ctx, &types.MsgCancelOrder{OrderId: sell.OrderId, GoodTilOneof: &types.MsgCancelOrder_GoodTilBlock{GoodTilBlock: sell.GetGoodTilBlock()}})
	require.NoError(t, err)
	_, found = memClob.spotCandidateLedger.Get(sell.OrderId)
	require.True(t, found)
}

func TestSpotCandidateNodesMayAcceptDifferentCompetingOrders(t *testing.T) {
	ctx, _, _ := sdktest.NewSdkContextWithMultistore()
	ctx = ctx.WithIsCheckTx(true).WithTxBytes([]byte{1, 2, 3})
	newNode := func() (*MemClobPriceTimePriority, types.Order, types.Order) {
		node, first, pair := newSpotCandidatePlacementFixture(t)
		second := first
		second.OrderId.ClientId++
		node.SetClobKeeper(memclob.NewFakeMemClobKeeper().WithSpotCandidateStateFn(
			func(_ sdk.Context, order types.Order) (types.SpotCandidateState, error) {
				return types.SpotCandidateState{
					ClobPair: pair, OutgoingAssetId: 0, TotalQuantums: big.NewInt(22),
					StatefulReservedQuantums: new(big.Int), CurrentOrderEpoch: order.OrderEpoch,
				}, nil
			},
		))
		return node, first, second
	}

	nodeA, firstA, secondA := newNode()
	_, _, _, err := nodeA.PlaceOrder(ctx, firstA)
	require.NoError(t, err)
	_, _, _, err = nodeA.PlaceOrder(ctx, secondA)
	require.ErrorIs(t, err, ErrInsufficientSpotBalance)

	nodeB, firstB, secondB := newNode()
	_, _, _, err = nodeB.PlaceOrder(ctx, secondB)
	require.NoError(t, err)
	_, _, _, err = nodeB.PlaceOrder(ctx, firstB)
	require.ErrorIs(t, err, ErrInsufficientSpotBalance)

	require.Equal(t, uint64(1), nodeA.spotCandidateLedger.OrderCount())
	require.Equal(t, uint64(1), nodeB.spotCandidateLedger.OrderCount())
	_, firstOnA := nodeA.spotCandidateLedger.Get(firstA.OrderId)
	_, firstOnB := nodeB.spotCandidateLedger.Get(firstB.OrderId)
	require.True(t, firstOnA)
	require.False(t, firstOnB)
}

func TestSpotCandidateCapacityIsNodeLocalAndRestartStartsEmpty(t *testing.T) {
	ctx, _, _ := sdktest.NewSdkContextWithMultistore()
	ctx = ctx.WithIsCheckTx(true).WithTxBytes([]byte{1})
	newNode := func(maxOrders uint64) (*MemClobPriceTimePriority, types.Order, types.Order) {
		node, first, pair := newSpotCandidatePlacementFixture(t)
		node.spotCandidateLedger = NewSpotCandidateLedger(maxOrders, 1_000)
		second := first
		second.OrderId.ClientId++
		node.SetClobKeeper(memclob.NewFakeMemClobKeeper().WithSpotCandidateStateFn(
			func(_ sdk.Context, order types.Order) (types.SpotCandidateState, error) {
				return types.SpotCandidateState{
					ClobPair: pair, OutgoingAssetId: 0, TotalQuantums: big.NewInt(100),
					StatefulReservedQuantums: new(big.Int), CurrentOrderEpoch: order.OrderEpoch,
				}, nil
			},
		))
		return node, first, second
	}

	small, first, second := newNode(1)
	large, _, _ := newNode(2)
	for _, node := range []*MemClobPriceTimePriority{small, large} {
		_, _, _, err := node.PlaceOrder(ctx, first)
		require.NoError(t, err)
	}
	_, _, _, err := small.PlaceOrder(ctx, second)
	require.ErrorIs(t, err, ErrSpotCandidateOrderLimit)
	_, _, _, err = large.PlaceOrder(ctx, second)
	require.NoError(t, err)
	require.Equal(t, uint64(1), small.spotCandidateLedger.OrderCount())
	require.Equal(t, uint64(2), large.spotCandidateLedger.OrderCount())

	restarted, resent, _ := newNode(1)
	require.Zero(t, restarted.spotCandidateLedger.OrderCount())
	_, _, _, err = restarted.PlaceOrder(ctx, resent)
	require.NoError(t, err)
}
