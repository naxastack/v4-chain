package memclob

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

func isSpotShortTermOrder(order types.Order) bool {
	return order.IsShortTermOrder() && (order.MaxTradingFeePpm != 0 || order.OrderEpoch != 0)
}

// reserveSpotCandidate validates the consensus snapshot and atomically installs
// the node-local reservation. The returned function restores the prior
// reservation for placement paths that do not retain an order or pending fill.
func (m *MemClobPriceTimePriority) reserveSpotCandidate(
	ctx sdk.Context,
	order types.Order,
) (restore func(), err error) {
	if !isSpotShortTermOrder(order) {
		return func() {}, nil
	}
	state, err := m.clobKeeper.GetSpotCandidateState(ctx, order)
	if err != nil {
		return nil, err
	}
	if order.OrderEpoch != state.CurrentOrderEpoch {
		return nil, errorsmod.Wrapf(
			types.ErrSpotOrderEpochMismatch,
			"order epoch %d does not match current epoch %d",
			order.OrderEpoch,
			state.CurrentOrderEpoch,
		)
	}
	remaining, hasRemaining := m.GetOrderRemainingAmount(ctx, order)
	if !hasRemaining {
		return nil, types.ErrOrderFullyFilled
	}
	reservation, err := NewSpotCandidateReservation(order, state.ClobPair, remaining, ctx.TxBytes(), false)
	if err != nil {
		return nil, err
	}
	if reservation.OutgoingAssetId != state.OutgoingAssetId {
		return nil, fmt.Errorf(
			"spot candidate outgoing asset mismatch: calculated %d, snapshot %d",
			reservation.OutgoingAssetId,
			state.OutgoingAssetId,
		)
	}
	previous, existed := m.spotCandidateLedger.Get(order.OrderId)
	if err := m.spotCandidateLedger.UpsertWithinAvailable(
		reservation,
		state.TotalQuantums,
		state.StatefulReservedQuantums,
	); err != nil {
		return nil, err
	}
	return func() {
		if existed {
			if err := m.spotCandidateLedger.Upsert(previous); err != nil {
				panic(fmt.Sprintf("failed to restore spot candidate reservation: %v", err))
			}
			return
		}
		m.spotCandidateLedger.Remove(order.OrderId)
	}, nil
}
