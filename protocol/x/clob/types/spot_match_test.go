package types_test

import (
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func spotMatchOrders(makerIsBuy bool) types.MatchWithOrders {
	maker := spotTestOrder()
	maker.OrderId.SubaccountId = satypes.SubaccountId{Owner: "maker", Number: 1}
	maker.Quantums, maker.Subticks, maker.MaxTradingFeePpm = 10, 200, 1_000
	taker := spotTestOrder()
	taker.OrderId.SubaccountId = satypes.SubaccountId{Owner: "taker", Number: 2}
	taker.OrderId.ClientId = 2
	taker.Quantums, taker.Subticks, taker.MaxTradingFeePpm = 10, 200, 1_000
	if makerIsBuy {
		maker.Side, taker.Side = types.Order_SIDE_BUY, types.Order_SIDE_SELL
	} else {
		maker.Side, taker.Side = types.Order_SIDE_SELL, types.Order_SIDE_BUY
	}
	return types.MatchWithOrders{MakerOrder: &maker, TakerOrder: &taker, FillAmount: satypes.BaseQuantums(10)}
}

func TestCalculateSpotMatchSettlement(t *testing.T) {
	for _, makerIsBuyer := range []bool{true, false} {
		t.Run(map[bool]string{true: "maker buyer", false: "taker buyer"}[makerIsBuyer], func(t *testing.T) {
			result, err := types.CalculateSpotMatchSettlement(spotMatchOrders(makerIsBuyer), spotTestPair(), 1_000)
			require.NoError(t, err)
			require.Equal(t, uint32(1), result.BaseAssetId)
			require.Equal(t, uint32(0), result.QuoteAssetId)
			require.Equal(t, big.NewInt(10), result.BaseQuantums)
			require.Equal(t, big.NewInt(2_000), result.QuoteQuantums)
			require.Equal(t, big.NewInt(2), result.BuyerFeeQuoteQuantums)
			require.Equal(t, big.NewInt(2), result.SellerFeeQuoteQuantums)
			require.Equal(t, big.NewInt(10), result.BuyerBaseDelta)
			require.Equal(t, big.NewInt(-2_002), result.BuyerQuoteDelta)
			require.Equal(t, big.NewInt(-10), result.SellerBaseDelta)
			require.Equal(t, big.NewInt(1_998), result.SellerQuoteDelta)
			require.Equal(t, makerIsBuyer, result.MakerIsBuyer)
		})
	}
}

func TestCalculateSpotMatchSettlementUsesMakerPriceAndRoundsFeesUp(t *testing.T) {
	match := spotMatchOrders(false)
	maker, taker := match.MakerOrder.MustGetOrder(), match.TakerOrder.MustGetOrder()
	maker.Subticks, taker.Subticks = 101, 102
	match.MakerOrder, match.TakerOrder = &maker, &taker
	result, err := types.CalculateSpotMatchSettlement(match, spotTestPair(), 1_000)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1_010), result.QuoteQuantums)
	require.Equal(t, big.NewInt(2), result.BuyerFeeQuoteQuantums)
}

func TestCalculateSpotMatchSettlementRejectsInvalidInputs(t *testing.T) {
	t.Run("fee cap exceeded", func(t *testing.T) {
		match := spotMatchOrders(true)
		maker := match.MakerOrder.MustGetOrder()
		maker.MaxTradingFeePpm = 999
		match.MakerOrder = &maker
		_, err := types.CalculateSpotMatchSettlement(match, spotTestPair(), 1_000)
		require.Error(t, err)
	})
	t.Run("zero quote", func(t *testing.T) {
		pair := spotTestPair()
		pair.QuantumConversionExponent = -4
		_, err := types.CalculateSpotMatchSettlement(spotMatchOrders(true), pair, 1_000)
		require.Error(t, err)
	})
	t.Run("seller net quote not positive", func(t *testing.T) {
		match := spotMatchOrders(true)
		maker, taker := match.MakerOrder.MustGetOrder(), match.TakerOrder.MustGetOrder()
		maker.MaxTradingFeePpm, taker.MaxTradingFeePpm = 1_000_000, 1_000_000
		match.MakerOrder, match.TakerOrder = &maker, &taker
		_, err := types.CalculateSpotMatchSettlement(match, spotTestPair(), 1_000_000)
		require.Error(t, err)
	})
	t.Run("perpetual pair", func(t *testing.T) {
		pair := spotTestPair()
		pair.Metadata = &types.ClobPair_PerpetualClobMetadata{}
		_, err := types.CalculateSpotMatchSettlement(spotMatchOrders(true), pair, 1_000)
		require.Error(t, err)
	})
}
