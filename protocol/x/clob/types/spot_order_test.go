package types_test

import (
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func spotTestOrder() types.Order {
	return types.Order{
		OrderId: types.OrderId{
			SubaccountId: satypes.SubaccountId{Owner: "dydx1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp8a3s", Number: 1},
			ClientId:     1,
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

func spotTestPair() types.ClobPair {
	return types.ClobPair{
		Id:                        7,
		Metadata:                  &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}},
		StepBaseQuantums:          5,
		SubticksPerTick:           1,
		QuantumConversionExponent: 0,
		Status:                    types.ClobPair_STATUS_ACTIVE,
	}
}

func TestValidateSpotLongTermOrder(t *testing.T) {
	valid := spotTestOrder()
	require.NoError(t, types.ValidateSpotLongTermOrder(valid, 1_000))

	tests := map[string]func(*types.Order){
		"short term":    func(order *types.Order) { order.OrderId.OrderFlags = types.OrderIdFlags_ShortTerm },
		"post only":     func(order *types.Order) { order.TimeInForce = types.Order_TIME_IN_FORCE_POST_ONLY },
		"reduce only":   func(order *types.Order) { order.ReduceOnly = true },
		"conditional":   func(order *types.Order) { order.ConditionType = types.Order_CONDITION_TYPE_STOP_LOSS },
		"twap":          func(order *types.Order) { order.TwapParameters = &types.TwapParameters{Duration: 300, Interval: 30} },
		"builder":       func(order *types.Order) { order.BuilderCodeParameters = &types.BuilderCodeParameters{} },
		"router":        func(order *types.Order) { order.OrderRouterAddress = "router" },
		"fee too low":   func(order *types.Order) { order.MaxTradingFeePpm = 999 },
		"fee too high":  func(order *types.Order) { order.MaxTradingFeePpm = 1_000_001 },
		"nonzero epoch": func(order *types.Order) { order.OrderEpoch = 1 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			order := spotTestOrder()
			mutate(&order)
			require.Error(t, types.ValidateSpotLongTermOrder(order, 1_000))
		})
	}
}

func TestValidatePerpetualOrderSpotFields(t *testing.T) {
	order := spotTestOrder()
	order.MaxTradingFeePpm = 0
	require.NoError(t, types.ValidatePerpetualOrderSpotFields(order))
	order.MaxTradingFeePpm = 1
	require.Error(t, types.ValidatePerpetualOrderSpotFields(order))
}

func TestCalculateSpotOrderReservation(t *testing.T) {
	pair := spotTestPair()
	buy := spotTestOrder()
	assetId, required, err := types.CalculateSpotOrderReservation(buy, pair)
	require.NoError(t, err)
	require.Equal(t, uint32(0), assetId)
	require.Equal(t, big.NewInt(22), required)

	sell := buy
	sell.Side = types.Order_SIDE_SELL
	assetId, required, err = types.CalculateSpotOrderReservation(sell, pair)
	require.NoError(t, err)
	require.Equal(t, uint32(1), assetId)
	require.Equal(t, big.NewInt(10), required)
}

func TestCalculateSpotOrderReservationRejectsInvalidInputs(t *testing.T) {
	pair := spotTestPair()
	order := spotTestOrder()

	order.Quantums = 11
	_, _, err := types.CalculateSpotOrderReservation(order, pair)
	require.Error(t, err)

	order = spotTestOrder()
	pair.QuantumConversionExponent = -2
	_, _, err = types.CalculateSpotOrderReservation(order, pair)
	require.Error(t, err)

	order = spotTestOrder()
	pair = spotTestPair()
	pair.Metadata = &types.ClobPair_PerpetualClobMetadata{PerpetualClobMetadata: &types.PerpetualClobMetadata{}}
	_, _, err = types.CalculateSpotOrderReservation(order, pair)
	require.Error(t, err)
}
