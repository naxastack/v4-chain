package cli

import (
	"fmt"

	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// SpotOrderCommandArgs is the operator input for a spot place or cancel command.
type SpotOrderCommandArgs struct {
	Owner            string
	SubaccountNumber uint32
	ClientId         uint32
	ClobPairId       uint32
	Side             int32
	Quantums         uint64
	Subticks         uint64
	GoodTil          uint32
	MaxTradingFeePpm uint32
	OrderEpoch       uint64
	LongTerm         bool
}

func newSpotPlaceOrderMsg(args SpotOrderCommandArgs) (*types.MsgPlaceOrder, error) {
	if args.Side != int32(types.Order_SIDE_BUY) && args.Side != int32(types.Order_SIDE_SELL) {
		return nil, fmt.Errorf("side must be %d (buy) or %d (sell)", types.Order_SIDE_BUY, types.Order_SIDE_SELL)
	}
	if args.Quantums == 0 {
		return nil, fmt.Errorf("quantums cannot be zero")
	}
	if args.Subticks == 0 {
		return nil, fmt.Errorf("subticks cannot be zero")
	}
	if args.GoodTil == 0 {
		return nil, fmt.Errorf("good til cannot be zero")
	}
	if args.MaxTradingFeePpm == 0 || args.MaxTradingFeePpm > uint32(lib.OneMillion) {
		return nil, fmt.Errorf("max trading fee ppm must be in 1..1000000")
	}

	order := types.Order{
		OrderId: types.OrderId{
			ClientId: args.ClientId,
			SubaccountId: satypes.SubaccountId{
				Owner:  args.Owner,
				Number: args.SubaccountNumber,
			},
			ClobPairId: args.ClobPairId,
			OrderFlags: types.OrderIdFlags_ShortTerm,
		},
		Side:             types.Order_Side(args.Side),
		Quantums:         args.Quantums,
		Subticks:         args.Subticks,
		GoodTilOneof:     &types.Order_GoodTilBlock{GoodTilBlock: args.GoodTil},
		MaxTradingFeePpm: args.MaxTradingFeePpm,
		OrderEpoch:       args.OrderEpoch,
	}
	if args.LongTerm {
		if args.OrderEpoch != 0 {
			return nil, fmt.Errorf("spot long-term order epoch must be zero")
		}
		order.OrderId.OrderFlags = types.OrderIdFlags_LongTerm
		order.GoodTilOneof = &types.Order_GoodTilBlockTime{GoodTilBlockTime: args.GoodTil}
		order.OrderEpoch = 0
	}
	return types.NewMsgPlaceOrder(order), nil
}

func newSpotCancelOrderMsg(args SpotOrderCommandArgs) (*types.MsgCancelOrder, error) {
	if args.GoodTil == 0 {
		return nil, fmt.Errorf("good til cannot be zero")
	}
	orderId := types.OrderId{
		ClientId: args.ClientId,
		SubaccountId: satypes.SubaccountId{
			Owner:  args.Owner,
			Number: args.SubaccountNumber,
		},
		ClobPairId: args.ClobPairId,
		OrderFlags: types.OrderIdFlags_ShortTerm,
	}
	if args.LongTerm {
		orderId.OrderFlags = types.OrderIdFlags_LongTerm
		return types.NewMsgCancelOrderStateful(orderId, args.GoodTil), nil
	}
	return types.NewMsgCancelOrderShortTerm(orderId, args.GoodTil), nil
}
