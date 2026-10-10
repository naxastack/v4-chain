package memclob

import (
	"errors"
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func spotCandidateTestOrderId(owner string, number, clientId uint32) types.OrderId {
	return types.OrderId{SubaccountId: satypes.SubaccountId{Owner: owner, Number: number}, ClientId: clientId, OrderFlags: types.OrderIdFlags_ShortTerm, ClobPairId: 1}
}

func spotCandidateTestReservation(orderId types.OrderId, assetId uint32, reserved int64, txBytes []byte) SpotCandidateReservation {
	return SpotCandidateReservation{OrderId: orderId, OutgoingAssetId: assetId, RemainingBaseQuantums: 10, ReservedQuantums: big.NewInt(reserved), OrderEpoch: 3, TxBytes: txBytes}
}

func TestSpotCandidateLedgerAggregatesAcrossOrders(t *testing.T) {
	ledger := NewSpotCandidateLedger(3, 20)
	firstId := spotCandidateTestOrderId("alice", 0, 1)
	secondId := spotCandidateTestOrderId("alice", 0, 2)
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(firstId, 0, 11, []byte{1, 2})))
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(secondId, 0, 17, []byte{3, 4, 5})))
	require.Equal(t, big.NewInt(28), ledger.GetReserved(firstId.SubaccountId, 0))
	require.Equal(t, uint64(2), ledger.OrderCount())
	require.Equal(t, uint64(5), ledger.CandidateBytes())
	require.True(t, ledger.Remove(firstId))
	require.Equal(t, big.NewInt(17), ledger.GetReserved(firstId.SubaccountId, 0))
	require.False(t, ledger.Remove(firstId))
}

func TestSpotCandidateLedgerReplacementUsesNetCapacity(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 4)
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(orderId, 0, 10, []byte{1, 2, 3, 4})))
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(orderId, 1, 7, []byte{5, 6, 7})))
	require.Zero(t, ledger.GetReserved(orderId.SubaccountId, 0).Sign())
	require.Equal(t, big.NewInt(7), ledger.GetReserved(orderId.SubaccountId, 1))
	require.Equal(t, uint64(3), ledger.CandidateBytes())
}

func TestSpotCandidateLedgerCapacityFailureIsAtomic(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 4)
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(orderId, 0, 10, []byte{1, 2})))
	err := ledger.Upsert(spotCandidateTestReservation(orderId, 1, 20, []byte{1, 2, 3, 4, 5}))
	require.ErrorIs(t, err, ErrSpotCandidateByteLimit)
	require.Equal(t, big.NewInt(10), ledger.GetReserved(orderId.SubaccountId, 0))
	require.Zero(t, ledger.GetReserved(orderId.SubaccountId, 1).Sign())
	require.Equal(t, uint64(2), ledger.CandidateBytes())
	err = ledger.Upsert(spotCandidateTestReservation(spotCandidateTestOrderId("alice", 0, 2), 0, 1, nil))
	require.ErrorIs(t, err, ErrSpotCandidateOrderLimit)
	require.Equal(t, uint64(1), ledger.OrderCount())
}

func TestSpotCandidateLedgerCloneIsIndependent(t *testing.T) {
	ledger := NewSpotCandidateLedger(2, 20)
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	txBytes := []byte{1, 2}
	reservation := spotCandidateTestReservation(orderId, 0, 10, txBytes)
	require.NoError(t, ledger.Upsert(reservation))
	txBytes[0] = 9
	reservation.ReservedQuantums.SetInt64(99)
	stored, found := ledger.Get(orderId)
	require.True(t, found)
	require.Equal(t, []byte{1, 2}, stored.TxBytes)
	require.Equal(t, big.NewInt(10), stored.ReservedQuantums)
	clone := ledger.Clone()
	require.True(t, clone.Remove(orderId))
	require.Equal(t, uint64(1), ledger.OrderCount())
	require.Equal(t, uint64(0), clone.OrderCount())
}

func TestSpotCandidateLedgerRejectsNonPositiveReservation(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 1)
	reservation := spotCandidateTestReservation(spotCandidateTestOrderId("alice", 0, 1), 0, 0, nil)
	require.True(t, errors.Is(ledger.Upsert(reservation), ErrInvalidSpotReservation))
	reservation.ReservedQuantums = nil
	require.True(t, errors.Is(ledger.Upsert(reservation), ErrInvalidSpotReservation))
}

func TestNewMemClobOwnsDefaultSpotCandidateLedger(t *testing.T) {
	memClob := NewMemClobPriceTimePriority(false)
	require.NotNil(t, memClob.spotCandidateLedger)
	require.Equal(t, uint64(0), memClob.spotCandidateLedger.OrderCount())
	require.Equal(t, DefaultMaxSpotShortTermOrders, memClob.spotCandidateLedger.maxOrders)
	require.Equal(t, DefaultMaxSpotShortTermCandidateBytes, memClob.spotCandidateLedger.maxBytes)
}

func TestNewSpotCandidateReservationUsesSharedBudget(t *testing.T) {
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	order := types.Order{
		OrderId:          orderId,
		Side:             types.Order_SIDE_BUY,
		Quantums:         10,
		Subticks:         2,
		MaxTradingFeePpm: 1_000,
		OrderEpoch:       7,
	}
	pair := types.ClobPair{
		Id:                        1,
		Metadata:                  &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}},
		StepBaseQuantums:          5,
		SubticksPerTick:           1,
		QuantumConversionExponent: 0,
	}

	reservation, err := NewSpotCandidateReservation(order, pair, 5, []byte{1, 2, 3}, true)
	require.NoError(t, err)
	require.Equal(t, orderId, reservation.OrderId)
	require.Equal(t, uint32(0), reservation.OutgoingAssetId)
	require.Equal(t, satypes.BaseQuantums(5), reservation.RemainingBaseQuantums)
	require.Equal(t, big.NewInt(11), reservation.ReservedQuantums)
	require.Equal(t, uint64(7), reservation.OrderEpoch)
	require.Equal(t, []byte{1, 2, 3}, reservation.TxBytes)
	require.True(t, reservation.PendingConfirmation)

	order.Side = types.Order_SIDE_SELL
	reservation, err = NewSpotCandidateReservation(order, pair, 5, []byte{4}, false)
	require.NoError(t, err)
	require.Equal(t, uint32(1), reservation.OutgoingAssetId)
	require.Equal(t, big.NewInt(5), reservation.ReservedQuantums)
	require.False(t, reservation.PendingConfirmation)
}

func TestNewSpotCandidateReservationRejectsZeroRemaining(t *testing.T) {
	order := types.Order{OrderId: spotCandidateTestOrderId("alice", 0, 1), Side: types.Order_SIDE_BUY, Quantums: 10, Subticks: 2, MaxTradingFeePpm: 1_000}
	pair := types.ClobPair{Id: 1, Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}}, StepBaseQuantums: 5, SubticksPerTick: 1}
	_, err := NewSpotCandidateReservation(order, pair, 0, nil, false)
	require.ErrorIs(t, err, ErrInvalidSpotReservation)
}

func TestSpotCandidateLedgerUpsertWithinAvailableAggregatesLocalReservations(t *testing.T) {
	ledger := NewSpotCandidateLedger(3, 20)
	firstId := spotCandidateTestOrderId("alice", 0, 1)
	secondId := spotCandidateTestOrderId("alice", 0, 2)
	require.NoError(t, ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(firstId, 0, 11, []byte{1}),
		big.NewInt(30),
		big.NewInt(2),
	))
	require.NoError(t, ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(secondId, 0, 17, []byte{2}),
		big.NewInt(30),
		big.NewInt(2),
	))
	require.Equal(t, big.NewInt(28), ledger.GetReserved(firstId.SubaccountId, 0))
}

func TestSpotCandidateLedgerUpsertWithinAvailableFailureIsAtomic(t *testing.T) {
	ledger := NewSpotCandidateLedger(2, 20)
	firstId := spotCandidateTestOrderId("alice", 0, 1)
	secondId := spotCandidateTestOrderId("alice", 0, 2)
	require.NoError(t, ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(firstId, 0, 11, []byte{1, 2}),
		big.NewInt(20),
		big.NewInt(2),
	))

	err := ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(secondId, 0, 8, []byte{3}),
		big.NewInt(20),
		big.NewInt(2),
	)
	require.ErrorIs(t, err, ErrInsufficientSpotBalance)
	require.Equal(t, uint64(1), ledger.OrderCount())
	require.Equal(t, uint64(2), ledger.CandidateBytes())
	require.Equal(t, big.NewInt(11), ledger.GetReserved(firstId.SubaccountId, 0))
	_, found := ledger.Get(secondId)
	require.False(t, found)
}

func TestSpotCandidateLedgerUpsertWithinAvailableReplacementUsesNetReservation(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 20)
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	require.NoError(t, ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(orderId, 0, 15, []byte{1, 2}),
		big.NewInt(20),
		big.NewInt(5),
	))
	require.NoError(t, ledger.UpsertWithinAvailable(
		spotCandidateTestReservation(orderId, 0, 10, []byte{3}),
		big.NewInt(20),
		big.NewInt(5),
	))
	require.Equal(t, big.NewInt(10), ledger.GetReserved(orderId.SubaccountId, 0))
	require.Equal(t, uint64(1), ledger.CandidateBytes())
}

func TestSpotCandidateLedgerUpsertWithinAvailableRejectsInvalidBalance(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 1)
	reservation := spotCandidateTestReservation(spotCandidateTestOrderId("alice", 0, 1), 0, 1, nil)
	require.ErrorIs(t, ledger.UpsertWithinAvailable(reservation, nil, big.NewInt(0)), ErrInvalidSpotBalance)
	require.ErrorIs(t, ledger.UpsertWithinAvailable(reservation, big.NewInt(0), big.NewInt(-1)), ErrInvalidSpotBalance)
	require.Zero(t, ledger.OrderCount())
}

func TestSpotCandidateLedgerResetDiscardsDerivedStateAndPreservesLimits(t *testing.T) {
	ledger := NewSpotCandidateLedger(1, 4)
	orderId := spotCandidateTestOrderId("alice", 0, 1)
	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(orderId, 0, 3, []byte{1, 2})))
	require.True(t, ledger.MarkPendingConfirmation(orderId))

	ledger.Reset()
	require.Zero(t, ledger.OrderCount())
	require.Zero(t, ledger.CandidateBytes())
	require.Zero(t, ledger.GetReserved(orderId.SubaccountId, 0).Sign())
	require.Equal(t, uint64(1), ledger.maxOrders)
	require.Equal(t, uint64(4), ledger.maxBytes)

	require.NoError(t, ledger.Upsert(spotCandidateTestReservation(orderId, 0, 4, []byte{1, 2, 3, 4})))
	require.Equal(t, big.NewInt(4), ledger.GetReserved(orderId.SubaccountId, 0))
}
