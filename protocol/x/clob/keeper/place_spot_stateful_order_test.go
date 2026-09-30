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

func TestPlaceSpotStatefulOrderReservationLifecycle(t *testing.T) {
	newContext := func(t *testing.T, balance int64) (keepertest.ClobKeepersTestContext, satypes.SubaccountId) {
		indexer := &mocks.IndexerEventManager{}
		indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		ks := keepertest.NewClobKeepersTestContext(
			t,
			memclob.NewMemClobPriceTimePriority(false),
			&mocks.BankKeeper{},
			indexer,
		)
		require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))
		require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
		_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
		require.NoError(t, err)
		for _, assetId := range []uint32{0, 1} {
			require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
				AssetId:            assetId,
				Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
				WithdrawalsEnabled: true,
				SpotTradingEnabled: true,
			}))
		}
		_, err = ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, spotReservationTestPair())
		require.NoError(t, err)
		id := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
		ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
			Id:          &id,
			AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
			AssetPositions: []*satypes.AssetPosition{{
				AssetId:                  assettypes.AssetUsdc.Id,
				Quantums:                 dtypes.NewInt(balance),
				StatefulReservedQuantums: dtypes.ZeroInt(),
			}},
		})
		ks.BlockTimeKeeper.SetPreviousBlockInfo(ks.Ctx, &blocktimetypes.BlockInfo{
			Timestamp: time.Unix(1, 0),
		})
		return ks, id
	}

	t.Run("success and duplicate", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)

		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))
		reservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, int64(22), reservation.ReservedQuantums.BigInt().Int64())
		require.Equal(t, uint64(10), reservation.RemainingBaseQuantums)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(22), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.True(t, found)

		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true)
		require.ErrorIs(t, err, types.ErrStatefulOrderAlreadyExists)
		stored = ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(22), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())

		require.NoError(t, ks.ClobKeeper.CancelStatefulOrder(ctx, types.NewMsgCancelOrderStateful(order.OrderId, 100)))
		_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.False(t, found)
		stored = ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
		require.Equal(t, uint64(1), ks.SubaccountsKeeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))
		require.ErrorIs(t, ks.ClobKeeper.ReleaseStatefulSpotOrder(ctx, order.OrderId), types.ErrStatefulOrderDoesNotExist)

		second := spotReservationTestOrder(id, 2)
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: second}, true))
		_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, second.OrderId)
		require.True(t, found)
		stored = ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(22), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	})

	t.Run("release failure preserves order and reservation", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))

		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		stored.AssetPositions[0].StatefulReservedQuantums = dtypes.NewInt(21)
		ks.SubaccountsKeeper.SetSubaccount(ctx, stored)

		err := ks.ClobKeeper.ReleaseStatefulSpotOrder(ctx, order.OrderId)
		require.ErrorIs(t, err, satypes.ErrStatefulReservedQuantumsInvalid)
		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.True(t, found)
		require.True(t, ks.ClobKeeper.HasStatefulSpotOrderMarketIndex(ctx, order.OrderId))
	})

	t.Run("expired order releases reservation through unified removal", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))

		expired := ks.ClobKeeper.RemoveExpiredStatefulOrders(ctx, time.Unix(101, 0))
		require.Equal(t, []types.OrderId{order.OrderId}, expired)
		for _, orderId := range expired {
			ks.ClobKeeper.MustRemoveStatefulOrder(ctx, orderId)
		}

		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.False(t, found)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	})

	t.Run("resize reduces only the reservation delta", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)
		order.Quantums = 15
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))

		require.NoError(t, ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 10))
		reservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, uint64(10), reservation.RemainingBaseQuantums)
		require.Equal(t, int64(22), reservation.ReservedQuantums.BigInt().Int64())
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(22), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
		require.Equal(t, uint64(1), ks.SubaccountsKeeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))

		require.NoError(t, ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 5))
		require.NoError(t, ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 5))
		reservation, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, int64(11), reservation.ReservedQuantums.BigInt().Int64())
		stored = ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(11), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())

		err := ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 10)
		require.Error(t, err)
		reservation, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, uint64(5), reservation.RemainingBaseQuantums)
		require.Equal(t, int64(11), reservation.ReservedQuantums.BigInt().Int64())

		require.NoError(t, ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 0))
		_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.False(t, found)
		stored = ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
		require.Equal(t, uint64(1), ks.SubaccountsKeeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))
	})

	t.Run("resize failure preserves all order state", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		stored.AssetPositions[0].StatefulReservedQuantums = dtypes.NewInt(5)
		ks.SubaccountsKeeper.SetSubaccount(ctx, stored)

		err := ks.ClobKeeper.ResizeStatefulSpotReservation(ctx, order.OrderId, 5)
		require.ErrorIs(t, err, satypes.ErrStatefulReservedQuantumsInvalid)
		reservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, uint64(10), reservation.RemainingBaseQuantums)
		require.Equal(t, int64(22), reservation.ReservedQuantums.BigInt().Int64())
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.True(t, found)
	})

	t.Run("insufficient balance leaves no partial state", func(t *testing.T) {
		ks, id := newContext(t, 21)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		order := spotReservationTestOrder(id, 1)

		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true)
		require.ErrorIs(t, err, satypes.ErrStatefulReservedQuantumsInvalid)
		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.False(t, found)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	})

	t.Run("imported reservation restores active order count", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		imported := spotReservationTestOrder(id, 1)
		ks.ClobKeeper.ImportStatefulSpotReservation(ctx, types.StatefulSpotReservation{
			OrderId:               imported.OrderId,
			OutgoingAssetId:       assettypes.AssetUsdc.Id,
			ReservedQuantums:      dtypes.NewInt(22),
			RemainingBaseQuantums: imported.Quantums,
		})
		params := types.DefaultSpotResourceParams()
		params.MaxSpotStatefulOrdersPerSubaccount = 1
		require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ctx, params))

		second := spotReservationTestOrder(id, 2)
		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: second}, true)
		require.ErrorContains(t, err, "spot stateful order limit exceeded")
		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, second.OrderId)
		require.False(t, found)
	})
	t.Run("sell order reserves base asset", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		account := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		account.AssetPositions = append(account.AssetPositions, &satypes.AssetPosition{
			AssetId:                  1,
			Quantums:                 dtypes.NewInt(20),
			StatefulReservedQuantums: dtypes.ZeroInt(),
		})
		ks.SubaccountsKeeper.SetSubaccount(ctx, account)
		order := spotReservationTestOrder(id, 1)
		order.Side = types.Order_SIDE_SELL

		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true))
		reservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.True(t, found)
		require.Equal(t, uint32(1), reservation.OutgoingAssetId)
		require.Equal(t, int64(10), reservation.ReservedQuantums.BigInt().Int64())
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		base := requireAssetPosition(t, stored, 1)
		require.Equal(t, int64(10), base.StatefulReservedQuantums.BigInt().Int64())
	})

	t.Run("wrong account type leaves no partial state", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		account := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		account.AccountType = satypes.AccountType_ACCOUNT_TYPE_PERPETUAL
		ks.SubaccountsKeeper.SetSubaccount(ctx, account)
		order := spotReservationTestOrder(id, 1)

		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true)
		require.ErrorContains(t, err, "spot orders require a spot subaccount")
		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	})

	t.Run("buy orders aggregate across pairs", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		_, err := ks.AssetsKeeper.CreateAsset(ctx, 2, "SECOND", "usecond", 0, false, 0, 0)
		require.NoError(t, err)
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicy{
			AssetId:            2,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
		secondPair := spotReservationTestPair()
		secondPair.Id = 8
		secondPair.Metadata = &types.ClobPair_SpotClobMetadata{
			SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 2, QuoteAssetId: 0},
		}
		_, err = ks.ClobKeeper.CreateSpotClobPair(ctx, secondPair)
		require.NoError(t, err)
		first := spotReservationTestOrder(id, 1)
		second := spotReservationTestOrder(id, 2)
		second.OrderId.ClobPairId = 8

		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: first}, true))
		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: second}, true))
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		quote := requireAssetPosition(t, stored, assettypes.AssetUsdc.Id)
		require.Equal(t, int64(44), quote.StatefulReservedQuantums.BigInt().Int64())
	})
	t.Run("disabled asset leaves no partial state", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		policy, found := ks.AssetsKeeper.GetAssetPolicy(ctx, 1)
		require.True(t, found)
		policy.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
		require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ctx, policy))
		order := spotReservationTestOrder(id, 1)

		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: order}, true)
		require.Error(t, err)
		_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, order.OrderId)
		require.False(t, found)
		_, found = ks.ClobKeeper.GetLongTermOrderPlacement(ctx, order.OrderId)
		require.False(t, found)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	})
	t.Run("active order limit leaves second order unchanged", func(t *testing.T) {
		ks, id := newContext(t, 100)
		ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
		params := types.DefaultSpotResourceParams()
		params.MaxSpotStatefulOrdersPerSubaccount = 1
		require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ctx, params))
		first := spotReservationTestOrder(id, 1)
		second := spotReservationTestOrder(id, 2)

		require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: first}, true))
		err := ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: second}, true)
		require.ErrorContains(t, err, "spot stateful order limit exceeded")
		_, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, second.OrderId)
		require.False(t, found)
		stored := ks.SubaccountsKeeper.GetSubaccount(ctx, id)
		require.Equal(t, int64(22), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	})
}

func requireAssetPosition(
	t *testing.T,
	subaccount satypes.Subaccount,
	assetId uint32,
) *satypes.AssetPosition {
	t.Helper()
	for _, position := range subaccount.AssetPositions {
		if position.AssetId == assetId {
			return position
		}
	}
	require.FailNow(t, "asset position not found", "asset id: %d", assetId)
	return nil
}
func spotReservationTestPair() types.ClobPair {
	return types.ClobPair{
		Id:                        7,
		Metadata:                  &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}},
		StepBaseQuantums:          5,
		SubticksPerTick:           1,
		QuantumConversionExponent: 0,
		Status:                    types.ClobPair_STATUS_ACTIVE,
	}
}

func spotReservationTestOrder(id satypes.SubaccountId, clientId uint32) types.Order {
	return types.Order{
		OrderId: types.OrderId{
			SubaccountId: id,
			ClientId:     clientId,
			OrderFlags:   types.OrderIdFlags_LongTerm,
			ClobPairId:   7,
		},
		Side:             types.Order_SIDE_BUY,
		Quantums:         10,
		Subticks:         2,
		GoodTilOneof:     &types.Order_GoodTilBlockTime{GoodTilBlockTime: 100},
		MaxTradingFeePpm: 1_000,
	}
}
