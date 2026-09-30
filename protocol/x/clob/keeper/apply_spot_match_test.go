package keeper_test

import (
	"errors"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	affiliatetypes "github.com/dydxprotocol/v4-chain/protocol/x/affiliates/types"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	blocktimetypes "github.com/dydxprotocol/v4-chain/protocol/x/blocktime/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	feetiertypes "github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProcessSpotSingleMatchAtomicSettlement(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bankError error
		wantOK    bool
	}{
		{name: "success", wantOK: true},
		{name: "bank failure rolls back all state", bankError: errors.New("bank failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bankKeeper := &mocks.BankKeeper{}
			bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(tc.bankError).Once()
			indexer := &mocks.IndexerEventManager{}
			indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			ks := keepertest.NewClobKeepersTestContext(
				t,
				memclob.NewMemClobPriceTimePriority(false),
				bankKeeper,
				indexer,
			)
			require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))
			require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
			_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
			require.NoError(t, err)
			for _, assetId := range []uint32{0, 1} {
				require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
					AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
					WithdrawalsEnabled: true, SpotTradingEnabled: true,
				}))
			}
			pair := spotReservationTestPair()
			_, err = ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
			require.NoError(t, err)
			require.NoError(t, ks.FeeTiersKeeper.SetSpotFeeParams(ks.Ctx, assettypesToSpotFeeParams(1_000)))

			buyerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
			sellerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 2}
			ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
				Id: &buyerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
				AssetPositions: []*satypes.AssetPosition{{
					AssetId: 0, Quantums: dtypes.NewInt(100), StatefulReservedQuantums: dtypes.ZeroInt(),
				}},
			})
			ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
				Id: &sellerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
				AssetPositions: []*satypes.AssetPosition{{
					AssetId: 1, Quantums: dtypes.NewInt(10), StatefulReservedQuantums: dtypes.ZeroInt(),
				}},
			})
			ks.BlockTimeKeeper.SetPreviousBlockInfo(ks.Ctx, &blocktimetypes.BlockInfo{Timestamp: time.Unix(1, 0)})
			ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
			buyerOrder := spotReservationTestOrder(buyerId, 1)
			sellerOrder := spotReservationTestOrder(sellerId, 2)
			sellerOrder.Side = types.Order_SIDE_SELL
			require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: buyerOrder}, true))
			require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: sellerOrder}, true))

			match := &types.MatchWithOrders{
				MakerOrder: &sellerOrder,
				TakerOrder: &buyerOrder,
				FillAmount: satypes.BaseQuantums(10),
			}
			success, _, _, affiliateShare, err := ks.ClobKeeper.ProcessSingleMatch(
				ctx,
				match,
				nil,
				affiliatetypes.AffiliateParameters{},
			)
			if !tc.wantOK {
				require.ErrorIs(t, err, tc.bankError)
				require.False(t, success)
				require.Equal(t, int64(100), requireAssetPosition(t, ks.SubaccountsKeeper.GetSubaccount(ctx, buyerId), 0).GetBigQuantums().Int64())
				require.Equal(t, int64(22), requireAssetPosition(t, ks.SubaccountsKeeper.GetSubaccount(ctx, buyerId), 0).StatefulReservedQuantums.BigInt().Int64())
				require.Equal(t, int64(10), requireAssetPosition(t, ks.SubaccountsKeeper.GetSubaccount(ctx, sellerId), 1).GetBigQuantums().Int64())
				require.Equal(t, int64(10), requireAssetPosition(t, ks.SubaccountsKeeper.GetSubaccount(ctx, sellerId), 1).StatefulReservedQuantums.BigInt().Int64())
				_, buyerFill, _ := ks.ClobKeeper.GetOrderFillAmount(ctx, buyerOrder.OrderId)
				_, sellerFill, _ := ks.ClobKeeper.GetOrderFillAmount(ctx, sellerOrder.OrderId)
				require.Zero(t, buyerFill)
				require.Zero(t, sellerFill)
				return
			}

			require.NoError(t, err)
			require.True(t, success)
			require.Zero(t, affiliateShare.Sign())
			require.Equal(t, int64(1), match.MakerFee)
			require.Equal(t, int64(1), match.TakerFee)
			buyer := ks.SubaccountsKeeper.GetSubaccount(ctx, buyerId)
			seller := ks.SubaccountsKeeper.GetSubaccount(ctx, sellerId)
			require.Equal(t, int64(79), requireAssetPosition(t, buyer, 0).GetBigQuantums().Int64())
			require.Equal(t, int64(10), requireAssetPosition(t, buyer, 1).GetBigQuantums().Int64())
			require.Equal(t, int64(19), requireAssetPosition(t, seller, 0).GetBigQuantums().Int64())
			sellerHasBasePosition := false
			for _, position := range seller.AssetPositions {
				if position.AssetId == 1 {
					sellerHasBasePosition = true
					require.Zero(t, position.GetBigQuantums().Sign())
				}
			}
			require.False(t, sellerHasBasePosition)
			require.Zero(t, requireAssetPosition(t, buyer, 0).StatefulReservedQuantums.BigInt().Sign())
			buyerReservation, found := ks.ClobKeeper.GetStatefulSpotReservation(ctx, buyerOrder.OrderId)
			require.True(t, found)
			require.Zero(t, buyerReservation.RemainingBaseQuantums)
			require.Zero(t, buyerReservation.ReservedQuantums.BigInt().Sign())
			_, buyerFill, _ := ks.ClobKeeper.GetOrderFillAmount(ctx, buyerOrder.OrderId)
			_, sellerFill, _ := ks.ClobKeeper.GetOrderFillAmount(ctx, sellerOrder.OrderId)
			require.Equal(t, satypes.BaseQuantums(10), buyerFill)
			require.Equal(t, satypes.BaseQuantums(10), sellerFill)

			ks.ClobKeeper.MustRemoveStatefulOrder(ctx, buyerOrder.OrderId)
			ks.ClobKeeper.MustRemoveStatefulOrder(ctx, sellerOrder.OrderId)
			_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, buyerOrder.OrderId)
			require.False(t, found)
			_, found = ks.ClobKeeper.GetStatefulSpotReservation(ctx, sellerOrder.OrderId)
			require.False(t, found)
		})
	}
}

// Keep the fee-tier type dependency local to this test fixture.
func assettypesToSpotFeeParams(ppm uint32) feetiertypes.SpotFeeParams {
	return feetiertypes.SpotFeeParams{TradingFeePpm: ppm}
}

type spotSettlementFixture struct {
	keepers     keepertest.ClobKeepersTestContext
	ctx         sdk.Context
	buyerId     satypes.SubaccountId
	sellerId    satypes.SubaccountId
	buyerOrder  types.Order
	sellerOrder types.Order
}

func newSpotSettlementFixture(t *testing.T, orderQuantums uint64, buyerBalance int64, sellerBalance int64) spotSettlementFixture {
	t.Helper()
	bankKeeper := &mocks.BankKeeper{}
	bankKeeper.On("SendCoins", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(t, memclob.NewMemClobPriceTimePriority(false), bankKeeper, indexer)
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	_, err = ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, spotReservationTestPair())
	require.NoError(t, err)
	require.NoError(t, ks.FeeTiersKeeper.SetSpotFeeParams(ks.Ctx, assettypesToSpotFeeParams(1_000)))
	buyerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 11}
	sellerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 12}
	ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
		Id: &buyerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId: 0, Quantums: dtypes.NewInt(buyerBalance), StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
		Id: &sellerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId: 1, Quantums: dtypes.NewInt(sellerBalance), StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	ks.BlockTimeKeeper.SetPreviousBlockInfo(ks.Ctx, &blocktimetypes.BlockInfo{Timestamp: time.Unix(1, 0)})
	ctx := ks.Ctx.WithIsCheckTx(false).WithIsReCheckTx(false).WithBlockHeight(10)
	buyerOrder := spotReservationTestOrder(buyerId, 11)
	buyerOrder.Quantums = orderQuantums
	sellerOrder := spotReservationTestOrder(sellerId, 12)
	sellerOrder.Quantums = orderQuantums
	sellerOrder.Side = types.Order_SIDE_SELL
	require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: buyerOrder}, true))
	require.NoError(t, ks.ClobKeeper.PlaceStatefulOrder(ctx, &types.MsgPlaceOrder{Order: sellerOrder}, true))
	return spotSettlementFixture{
		keepers: ks, ctx: ctx, buyerId: buyerId, sellerId: sellerId,
		buyerOrder: buyerOrder, sellerOrder: sellerOrder,
	}
}

func TestProcessSpotSingleMatchPartialFillWithBuyerAsMaker(t *testing.T) {
	f := newSpotSettlementFixture(t, 15, 100, 15)
	match := &types.MatchWithOrders{
		MakerOrder: &f.buyerOrder,
		TakerOrder: &f.sellerOrder,
		FillAmount: satypes.BaseQuantums(5),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.NoError(t, err)
	require.True(t, success)
	buyer := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	seller := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.sellerId)
	require.Equal(t, int64(89), requireAssetPosition(t, buyer, 0).GetBigQuantums().Int64())
	require.Equal(t, int64(5), requireAssetPosition(t, buyer, 1).GetBigQuantums().Int64())
	require.Equal(t, int64(9), requireAssetPosition(t, seller, 0).GetBigQuantums().Int64())
	require.Equal(t, int64(10), requireAssetPosition(t, seller, 1).GetBigQuantums().Int64())
	require.Equal(t, int64(22), requireAssetPosition(t, buyer, 0).StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, int64(10), requireAssetPosition(t, seller, 1).StatefulReservedQuantums.BigInt().Int64())
	buyerReservation, found := f.keepers.ClobKeeper.GetStatefulSpotReservation(f.ctx, f.buyerOrder.OrderId)
	require.True(t, found)
	require.Equal(t, uint64(10), buyerReservation.RemainingBaseQuantums)
	require.Equal(t, int64(22), buyerReservation.ReservedQuantums.BigInt().Int64())
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Equal(t, satypes.BaseQuantums(5), buyerFill)
	require.Equal(t, satypes.BaseQuantums(5), sellerFill)
}

func TestProcessSpotSingleMatchRejectsDisabledAssetWithoutStateChanges(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	policy, found := f.keepers.AssetsKeeper.GetAssetPolicy(f.ctx, 1)
	require.True(t, found)
	policy.SpotTradingEnabled = false
	require.NoError(t, f.keepers.AssetsKeeper.UpdateAssetPolicy(f.ctx, policy))
	match := &types.MatchWithOrders{
		MakerOrder: &f.sellerOrder,
		TakerOrder: &f.buyerOrder,
		FillAmount: satypes.BaseQuantums(10),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.ErrorIs(t, err, assettypes.ErrAssetSpotTradingDisabled)
	require.False(t, success)
	require.Equal(t, int64(100), requireAssetPosition(t, f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId), 0).GetBigQuantums().Int64())
	require.Equal(t, int64(22), requireAssetPosition(t, f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId), 0).StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, int64(10), requireAssetPosition(t, f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.sellerId), 1).GetBigQuantums().Int64())
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func TestProcessSpotSingleMatchRejectsInsufficientBalanceWithoutStateChanges(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	buyer := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	requireAssetPosition(t, buyer, 0).Quantums = dtypes.NewInt(5)
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, buyer)
	match := &types.MatchWithOrders{
		MakerOrder: &f.sellerOrder,
		TakerOrder: &f.buyerOrder,
		FillAmount: satypes.BaseQuantums(10),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.ErrorIs(t, err, satypes.ErrBusinessAssetPositionNegative)
	require.False(t, success)
	buyer = f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	require.Equal(t, int64(5), requireAssetPosition(t, buyer, 0).GetBigQuantums().Int64())
	require.Equal(t, int64(22), requireAssetPosition(t, buyer, 0).StatefulReservedQuantums.BigInt().Int64())
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func TestProcessSpotSingleMatchRejectsOverfillWithoutStateChanges(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	match := &types.MatchWithOrders{
		MakerOrder: &f.sellerOrder,
		TakerOrder: &f.buyerOrder,
		FillAmount: satypes.BaseQuantums(15),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.Error(t, err)
	require.False(t, success)
	require.Equal(t, int64(100), requireAssetPosition(t, f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId), 0).GetBigQuantums().Int64())
	require.Equal(t, int64(10), requireAssetPosition(t, f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.sellerId), 1).GetBigQuantums().Int64())
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func TestProcessInternalOperationsSkipsRecoverableSpotFailure(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	buyer := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	requireAssetPosition(t, buyer, 0).Quantums = dtypes.NewInt(5)
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, buyer)

	operation := types.NewMatchOrdersInternalOperation(
		f.buyerOrder,
		[]types.MakerFill{{MakerOrderId: f.sellerOrder.OrderId, FillAmount: 10}},
	)
	results, err := f.keepers.ClobKeeper.ProcessInternalOperationsWithResults(
		f.ctx,
		[]types.InternalOperation{operation},
	)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, types.OperationExecutionStatusSkipped, results[0].Status)
	require.Equal(t, types.SpotOperationSkipReasonInsufficientBalance, results[0].SkipReason)

	skippedEventCount := 0
	for _, event := range f.ctx.EventManager().Events() {
		if event.Type == types.EventTypeSpotOperationSkipped {
			skippedEventCount++
		}
	}
	require.Equal(t, 1, skippedEventCount)

	buyer = f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	require.Equal(t, int64(5), requireAssetPosition(t, buyer, 0).GetBigQuantums().Int64())
	require.Equal(t, int64(22), requireAssetPosition(t, buyer, 0).StatefulReservedQuantums.BigInt().Int64())
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func TestProcessInternalOperationsContinuesAfterSkippedSpotMatch(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	firstBuyer := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	requireAssetPosition(t, firstBuyer, 0).Quantums = dtypes.NewInt(5)
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, firstBuyer)

	secondBuyerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 21}
	secondSellerId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 22}
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, satypes.Subaccount{
		Id: &secondBuyerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId: 0, Quantums: dtypes.NewInt(100), StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, satypes.Subaccount{
		Id: &secondSellerId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId: 1, Quantums: dtypes.NewInt(10), StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	secondBuyerOrder := spotReservationTestOrder(secondBuyerId, 21)
	secondSellerOrder := spotReservationTestOrder(secondSellerId, 22)
	secondSellerOrder.Side = types.Order_SIDE_SELL
	require.NoError(t, f.keepers.ClobKeeper.PlaceStatefulOrder(
		f.ctx, &types.MsgPlaceOrder{Order: secondBuyerOrder}, true,
	))
	require.NoError(t, f.keepers.ClobKeeper.PlaceStatefulOrder(
		f.ctx, &types.MsgPlaceOrder{Order: secondSellerOrder}, true,
	))

	operations := []types.InternalOperation{
		types.NewMatchOrdersInternalOperation(
			f.buyerOrder,
			[]types.MakerFill{{MakerOrderId: f.sellerOrder.OrderId, FillAmount: 10}},
		),
		types.NewMatchOrdersInternalOperation(
			secondBuyerOrder,
			[]types.MakerFill{{MakerOrderId: secondSellerOrder.OrderId, FillAmount: 10}},
		),
	}
	results, err := f.keepers.ClobKeeper.ProcessInternalOperationsWithResults(f.ctx, operations)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, types.OperationExecutionStatusSkipped, results[0].Status)
	require.Equal(t, types.SpotOperationSkipReasonInsufficientBalance, results[0].SkipReason)
	require.Equal(t, types.OperationExecutionStatusApplied, results[1].Status)
	require.Empty(t, results[1].SkipReason)

	_, firstBuyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.buyerOrder.OrderId)
	_, firstSellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	_, secondBuyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, secondBuyerOrder.OrderId)
	_, secondSellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, secondSellerOrder.OrderId)
	require.Zero(t, firstBuyerFill)
	require.Zero(t, firstSellerFill)
	require.Equal(t, satypes.BaseQuantums(10), secondBuyerFill)
	require.Equal(t, satypes.BaseQuantums(10), secondSellerFill)

	matchEvents := f.keepers.ClobKeeper.GenerateProcessProposerMatchesEvents(
		f.ctx,
		[]types.InternalOperation{results[1].Operation},
	)
	require.ElementsMatch(t, []types.OrderId{
		secondBuyerOrder.OrderId,
		secondSellerOrder.OrderId,
	}, matchEvents.OrderIdsFilledInLastBlock)
}

func TestProcessSpotSingleMatchOrderTypeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		makerShort bool
		takerShort bool
	}{
		{name: "long maker long taker"},
		{name: "long maker short taker", takerShort: true},
		{name: "short maker long taker", makerShort: true},
		{name: "short maker short taker", makerShort: true, takerShort: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newSpotSettlementFixture(t, 10, 100, 10)
			makerOrder := f.sellerOrder
			takerOrder := f.buyerOrder
			if tc.makerShort {
				f.keepers.ClobKeeper.MustRemoveStatefulOrder(f.ctx, makerOrder.OrderId)
				makerOrder = makeSpotShortTermOrder(f, makerOrder)
			}
			if tc.takerShort {
				f.keepers.ClobKeeper.MustRemoveStatefulOrder(f.ctx, takerOrder.OrderId)
				takerOrder = makeSpotShortTermOrder(f, takerOrder)
			}

			match := &types.MatchWithOrders{
				MakerOrder: &makerOrder,
				TakerOrder: &takerOrder,
				FillAmount: satypes.BaseQuantums(10),
			}
			success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
				f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
			)
			require.NoError(t, err)
			require.True(t, success)
			_, makerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, makerOrder.OrderId)
			_, takerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, takerOrder.OrderId)
			require.Equal(t, satypes.BaseQuantums(10), makerFill)
			require.Equal(t, satypes.BaseQuantums(10), takerFill)
		})
	}
}

func TestProcessSpotSingleMatchRejectsStaleShortTermOrderEpoch(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	f.keepers.ClobKeeper.MustRemoveStatefulOrder(f.ctx, f.buyerOrder.OrderId)
	buyerOrder := makeSpotShortTermOrder(f, f.buyerOrder)
	currentEpoch := f.keepers.SubaccountsKeeper.GetSpotOrderEpoch(f.ctx, f.buyerId, assettypes.AssetUsdc.Id)
	f.keepers.SubaccountsKeeper.SetSpotOrderEpoch(f.ctx, f.buyerId, assettypes.AssetUsdc.Id, currentEpoch+1)

	match := &types.MatchWithOrders{
		MakerOrder: &f.sellerOrder,
		TakerOrder: &buyerOrder,
		FillAmount: satypes.BaseQuantums(10),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.ErrorIs(t, err, types.ErrSpotOrderEpochMismatch)
	require.False(t, success)
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func makeSpotShortTermOrder(f spotSettlementFixture, order types.Order) types.Order {
	order.OrderId.OrderFlags = types.OrderIdFlags_ShortTerm
	order.GoodTilOneof = &types.Order_GoodTilBlock{GoodTilBlock: 20}
	pair := spotReservationTestPair()
	assetId := pair.GetSpotClobMetadata().QuoteAssetId
	if !order.IsBuy() {
		assetId = pair.GetSpotClobMetadata().BaseAssetId
	}
	order.OrderEpoch = f.keepers.SubaccountsKeeper.GetSpotOrderEpoch(
		f.ctx,
		order.OrderId.SubaccountId,
		assetId,
	)
	return order
}

func TestProcessInternalOperationsSkipsStaleSpotShortTermOrder(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	f.keepers.ClobKeeper.MustRemoveStatefulOrder(f.ctx, f.buyerOrder.OrderId)
	buyerOrder := makeSpotShortTermOrder(f, f.buyerOrder)
	currentEpoch := f.keepers.SubaccountsKeeper.GetSpotOrderEpoch(f.ctx, f.buyerId, assettypes.AssetUsdc.Id)
	f.keepers.SubaccountsKeeper.SetSpotOrderEpoch(f.ctx, f.buyerId, assettypes.AssetUsdc.Id, currentEpoch+1)

	operations := []types.InternalOperation{
		types.NewShortTermOrderPlacementInternalOperation(buyerOrder),
		types.NewMatchOrdersInternalOperation(
			buyerOrder,
			[]types.MakerFill{{MakerOrderId: f.sellerOrder.OrderId, FillAmount: 10}},
		),
	}
	results, err := f.keepers.ClobKeeper.ProcessInternalOperationsWithResults(f.ctx, operations)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, types.OperationExecutionStatusApplied, results[0].Status)
	require.Equal(t, types.OperationExecutionStatusSkipped, results[1].Status)
	require.Equal(t, types.SpotOperationSkipReasonOrderEpochMismatch, results[1].SkipReason)
	_, buyerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, buyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, buyerFill)
	require.Zero(t, sellerFill)
}

func TestSpotShortTermOrderCannotConsumeStatefulReservation(t *testing.T) {
	f := newSpotSettlementFixture(t, 10, 100, 10)
	buyer := f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	requireAssetPosition(t, buyer, assettypes.AssetUsdc.Id).Quantums = dtypes.NewInt(25)
	f.keepers.SubaccountsKeeper.SetSubaccount(f.ctx, buyer)

	shortBuyerOrder := f.buyerOrder
	shortBuyerOrder.OrderId.ClientId = 31
	shortBuyerOrder = makeSpotShortTermOrder(f, shortBuyerOrder)
	match := &types.MatchWithOrders{
		MakerOrder: &f.sellerOrder,
		TakerOrder: &shortBuyerOrder,
		FillAmount: satypes.BaseQuantums(10),
	}
	success, _, _, _, err := f.keepers.ClobKeeper.ProcessSingleMatch(
		f.ctx, match, nil, affiliatetypes.AffiliateParameters{},
	)
	require.ErrorIs(t, err, satypes.ErrStatefulReservedQuantumsInvalid)
	require.False(t, success)

	buyer = f.keepers.SubaccountsKeeper.GetSubaccount(f.ctx, f.buyerId)
	quotePosition := requireAssetPosition(t, buyer, assettypes.AssetUsdc.Id)
	require.Equal(t, int64(25), quotePosition.GetBigQuantums().Int64())
	require.Equal(t, int64(22), quotePosition.StatefulReservedQuantums.BigInt().Int64())
	_, shortFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, shortBuyerOrder.OrderId)
	_, sellerFill, _ := f.keepers.ClobKeeper.GetOrderFillAmount(f.ctx, f.sellerOrder.OrderId)
	require.Zero(t, shortFill)
	require.Zero(t, sellerFill)
}
