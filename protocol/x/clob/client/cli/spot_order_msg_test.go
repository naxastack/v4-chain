package cli

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func spotTestOwner() string {
	return constants.AliceAccAddress.String()
}

func TestNewSpotPlaceOrderMsgShortTerm(t *testing.T) {
	msg, err := newSpotPlaceOrderMsg(SpotOrderCommandArgs{
		Owner:            spotTestOwner(),
		SubaccountNumber: 1,
		ClientId:         7,
		ClobPairId:       100,
		Side:             int32(types.Order_SIDE_BUY),
		Quantums:         10,
		Subticks:         20,
		GoodTil:          30,
		MaxTradingFeePpm: 1_000,
		OrderEpoch:       4,
	})
	require.NoError(t, err)
	require.Equal(t, types.OrderIdFlags_ShortTerm, msg.Order.OrderId.OrderFlags)
	require.Equal(t, uint32(30), msg.Order.GetGoodTilBlock())
	require.Equal(t, uint64(4), msg.Order.OrderEpoch)
	require.Equal(t, uint32(1_000), msg.Order.MaxTradingFeePpm)
	require.NoError(t, msg.ValidateBasic())
}

func TestNewSpotPlaceOrderMsgLongTerm(t *testing.T) {
	msg, err := newSpotPlaceOrderMsg(SpotOrderCommandArgs{
		Owner:            spotTestOwner(),
		SubaccountNumber: 2,
		ClientId:         8,
		ClobPairId:       100,
		Side:             int32(types.Order_SIDE_SELL),
		Quantums:         11,
		Subticks:         21,
		GoodTil:          1_700_000_000,
		MaxTradingFeePpm: 2_000,
		LongTerm:         true,
	})
	require.NoError(t, err)
	require.Equal(t, types.OrderIdFlags_LongTerm, msg.Order.OrderId.OrderFlags)
	require.Equal(t, uint32(1_700_000_000), msg.Order.GetGoodTilBlockTime())
	require.Zero(t, msg.Order.OrderEpoch)
	require.NoError(t, msg.ValidateBasic())
}

func TestNewSpotPlaceOrderMsgRejectsLongTermEpoch(t *testing.T) {
	_, err := newSpotPlaceOrderMsg(SpotOrderCommandArgs{
		Owner:            spotTestOwner(),
		Side:             int32(types.Order_SIDE_BUY),
		Quantums:         1,
		Subticks:         1,
		GoodTil:          1,
		MaxTradingFeePpm: 1,
		OrderEpoch:       3,
		LongTerm:         true,
	})
	require.Error(t, err)
}

func TestNewSpotCancelOrderMsg(t *testing.T) {
	shortTerm, err := newSpotCancelOrderMsg(SpotOrderCommandArgs{
		Owner:            spotTestOwner(),
		SubaccountNumber: 1,
		ClientId:         7,
		ClobPairId:       100,
		GoodTil:          40,
	})
	require.NoError(t, err)
	require.Equal(t, uint32(40), shortTerm.GetGoodTilBlock())
	require.NoError(t, shortTerm.ValidateBasic())

	longTerm, err := newSpotCancelOrderMsg(SpotOrderCommandArgs{
		Owner:            spotTestOwner(),
		SubaccountNumber: 1,
		ClientId:         7,
		ClobPairId:       100,
		GoodTil:          50,
		LongTerm:         true,
	})
	require.NoError(t, err)
	require.Equal(t, types.OrderIdFlags_LongTerm, longTerm.OrderId.OrderFlags)
	require.Equal(t, uint32(50), longTerm.GetGoodTilBlockTime())
	require.NoError(t, longTerm.ValidateBasic())
}
