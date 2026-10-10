package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/indexer/indexer_manager"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	testtx "github.com/dydxprotocol/v4-chain/protocol/testutil/tx"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func TestDecodeShortTermOrderFromTxBytes(t *testing.T) {
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer_manager.NewIndexerEventManagerNoop(),
	)

	order := constants.Order_Alice_Num0_Id0_Clob0_Buy10_Price10_GTB16
	order.MaxTradingFeePpm = 1_000
	order.OrderEpoch = 7
	txBytes := testtx.MustGetTxBytes(types.NewMsgPlaceOrder(order))

	decoded, err := ks.ClobKeeper.DecodeShortTermOrderFromTxBytes(txBytes, order)
	require.NoError(t, err)
	require.Equal(t, order, decoded.Order)
}

func TestDecodeShortTermOrderFromTxBytesRejectsInvalidPayload(t *testing.T) {
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer_manager.NewIndexerEventManagerNoop(),
	)

	order := constants.Order_Alice_Num0_Id0_Clob0_Buy10_Price10_GTB16
	order.MaxTradingFeePpm = 1_000
	order.OrderEpoch = 7
	txBytes := testtx.MustGetTxBytes(types.NewMsgPlaceOrder(order))

	tests := map[string]struct {
		txBytes  []byte
		expected types.Order
	}{
		"missing transaction bytes": {
			expected: order,
		},
		"malformed transaction bytes": {
			txBytes:  []byte{0xff},
			expected: order,
		},
		"modified order epoch": {
			txBytes: txBytes,
			expected: func() types.Order {
				modified := order
				modified.OrderEpoch++
				return modified
			}(),
		},
		"modified maximum trading fee": {
			txBytes: txBytes,
			expected: func() types.Order {
				modified := order
				modified.MaxTradingFeePpm++
				return modified
			}(),
		},
		"multiple place-order messages": {
			txBytes: testtx.MustGetTxBytes(
				types.NewMsgPlaceOrder(order),
				types.NewMsgPlaceOrder(order),
			),
			expected: order,
		},
		"stateful order transaction": {
			txBytes: testtx.MustGetTxBytes(types.NewMsgPlaceOrder(
				constants.LongTermOrder_Bob_Num0_Id2_Clob0_Buy15_Price5_GTBT10,
			)),
			expected: order,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ks.ClobKeeper.DecodeShortTermOrderFromTxBytes(tc.txBytes, tc.expected)
			require.ErrorIs(t, err, types.ErrInvalidPlaceOrder)
		})
	}
}
