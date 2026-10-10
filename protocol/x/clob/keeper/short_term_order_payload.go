package keeper

import (
	errorsmod "cosmossdk.io/errors"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// DecodeShortTermOrderFromTxBytes decodes the original transaction and returns
// its single short-term order after verifying that it exactly matches the order
// retained in the local operations queue.
func (k Keeper) DecodeShortTermOrderFromTxBytes(
	txBytes []byte,
	expectedOrder types.Order,
) (*types.MsgPlaceOrder, error) {
	if len(txBytes) == 0 {
		return nil, errorsmod.Wrap(types.ErrInvalidPlaceOrder, "short-term order transaction bytes are empty")
	}

	tx, err := k.txDecoder(txBytes)
	if err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidPlaceOrder, "failed to decode short-term order transaction")
	}

	var decoded *types.MsgPlaceOrder
	for _, msg := range tx.GetMsgs() {
		placeOrder, ok := msg.(*types.MsgPlaceOrder)
		if !ok {
			continue
		}
		if decoded != nil {
			return nil, errorsmod.Wrap(types.ErrInvalidPlaceOrder, "transaction contains multiple place-order messages")
		}
		decoded = placeOrder
	}

	if decoded == nil || !decoded.Order.OrderId.IsShortTermOrder() {
		return nil, errorsmod.Wrap(types.ErrInvalidPlaceOrder, "transaction does not contain a short-term order")
	}
	if !proto.Equal(&decoded.Order, &expectedOrder) {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidPlaceOrder,
			"decoded short-term order payload does not match queued order: expected %X, got %X",
			expectedOrder.GetOrderHash(),
			decoded.Order.GetOrderHash(),
		)
	}

	return decoded, nil
}
