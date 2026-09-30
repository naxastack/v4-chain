package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateSpotOrderReservationForRemaining(t *testing.T) {
	pair := spotTestPair()
	order := spotTestOrder()
	order.Quantums = 15

	assetId, required, err := types.CalculateSpotOrderReservationForRemaining(order, pair, 10)
	require.NoError(t, err)
	require.Equal(t, uint32(0), assetId)
	require.Equal(t, int64(22), required.Int64())

	assetId, required, err = types.CalculateSpotOrderReservationForRemaining(order, pair, 5)
	require.NoError(t, err)
	require.Equal(t, uint32(0), assetId)
	require.Equal(t, int64(11), required.Int64())

	assetId, required, err = types.CalculateSpotOrderReservationForRemaining(order, pair, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), assetId)
	require.Zero(t, required.Sign())

	order.Side = types.Order_SIDE_SELL
	assetId, required, err = types.CalculateSpotOrderReservationForRemaining(order, pair, 5)
	require.NoError(t, err)
	require.Equal(t, uint32(1), assetId)
	require.Equal(t, int64(5), required.Int64())

	_, _, err = types.CalculateSpotOrderReservationForRemaining(order, pair, 7)
	require.Error(t, err)
}
