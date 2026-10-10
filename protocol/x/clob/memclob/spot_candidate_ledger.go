package memclob

import (
	"errors"
	"math/big"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

const (
	DefaultMaxSpotShortTermOrders         = uint64(100_000)
	DefaultMaxSpotShortTermCandidateBytes = uint64(64 << 20)
)

var (
	ErrSpotCandidateOrderLimit = errors.New("spot short-term candidate order limit exceeded")
	ErrSpotCandidateByteLimit  = errors.New("spot short-term candidate byte limit exceeded")
	ErrInvalidSpotReservation  = errors.New("spot short-term reservation must be positive")
	ErrInsufficientSpotBalance = errors.New("insufficient available balance for spot short-term candidate")
	ErrInvalidSpotBalance      = errors.New("spot balance inputs must be non-negative")
)

// SpotCandidateReservation is the node-local payment reservation for one spot
// short-term order. It is never persisted or used as a consensus input.
type SpotCandidateReservation struct {
	OrderId               types.OrderId
	OutgoingAssetId       uint32
	RemainingBaseQuantums satypes.BaseQuantums
	ReservedQuantums      *big.Int
	OrderEpoch            uint64
	TxBytes               []byte
	PendingConfirmation   bool
}

// NewSpotCandidateReservation calculates the node-local payment requirement
// using the same integer budget function as stateful spot reservations.
func NewSpotCandidateReservation(
	order types.Order,
	pair types.ClobPair,
	remainingBaseQuantums satypes.BaseQuantums,
	txBytes []byte,
	pendingConfirmation bool,
) (SpotCandidateReservation, error) {
	assetId, required, err := types.CalculateSpotOrderReservationForRemaining(
		order,
		pair,
		remainingBaseQuantums.ToUint64(),
	)
	if err != nil {
		return SpotCandidateReservation{}, err
	}
	if required.Sign() <= 0 {
		return SpotCandidateReservation{}, ErrInvalidSpotReservation
	}
	return SpotCandidateReservation{
		OrderId:               order.OrderId,
		OutgoingAssetId:       assetId,
		RemainingBaseQuantums: remainingBaseQuantums,
		ReservedQuantums:      required,
		OrderEpoch:            order.OrderEpoch,
		TxBytes:               append([]byte(nil), txBytes...),
		PendingConfirmation:   pendingConfirmation,
	}, nil
}

type spotCandidateBalanceKey struct {
	SubaccountId satypes.SubaccountId
	AssetId      uint32
}

// SpotCandidateLedger tracks node-local spot short-term reservations and their
// aggregate resource usage. Mutations validate their net effect before changing
// any state, so callers can compose them with a wider MemCLOB snapshot.
type SpotCandidateLedger struct {
	reservations map[types.OrderId]SpotCandidateReservation
	reserved     map[spotCandidateBalanceKey]*big.Int
	bytes        uint64
	maxOrders    uint64
	maxBytes     uint64
}

func NewSpotCandidateLedger(maxOrders, maxBytes uint64) *SpotCandidateLedger {
	return &SpotCandidateLedger{
		reservations: make(map[types.OrderId]SpotCandidateReservation),
		reserved:     make(map[spotCandidateBalanceKey]*big.Int),
		maxOrders:    maxOrders,
		maxBytes:     maxBytes,
	}
}

func NewDefaultSpotCandidateLedger() *SpotCandidateLedger {
	return NewSpotCandidateLedger(DefaultMaxSpotShortTermOrders, DefaultMaxSpotShortTermCandidateBytes)
}

func (l *SpotCandidateLedger) Clone() *SpotCandidateLedger {
	clone := NewSpotCandidateLedger(l.maxOrders, l.maxBytes)
	clone.bytes = l.bytes
	for orderId, reservation := range l.reservations {
		clone.reservations[orderId] = cloneSpotCandidateReservation(reservation)
	}
	for key, reserved := range l.reserved {
		clone.reserved[key] = new(big.Int).Set(reserved)
	}
	return clone
}

// Upsert inserts or replaces a reservation. Capacity is charged for retained
// transaction bytes and replacement checks use only the net resource change.
func (l *SpotCandidateLedger) Upsert(reservation SpotCandidateReservation) error {
	if reservation.ReservedQuantums == nil || reservation.ReservedQuantums.Sign() <= 0 {
		return ErrInvalidSpotReservation
	}
	previous, exists := l.reservations[reservation.OrderId]
	orderCount := uint64(len(l.reservations))
	if !exists {
		orderCount++
	}
	if orderCount > l.maxOrders {
		return ErrSpotCandidateOrderLimit
	}
	previousBytes := uint64(0)
	if exists {
		previousBytes = uint64(len(previous.TxBytes))
	}
	newBytes := uint64(len(reservation.TxBytes))
	if newBytes > l.maxBytes || l.bytes-previousBytes > l.maxBytes-newBytes {
		return ErrSpotCandidateByteLimit
	}
	if exists {
		l.subtract(previous)
	}
	stored := cloneSpotCandidateReservation(reservation)
	l.reservations[reservation.OrderId] = stored
	l.add(stored)
	l.bytes = l.bytes - previousBytes + newBytes
	return nil
}

// UpsertWithinAvailable atomically validates both local capacity and the
// account's available outgoing asset balance. totalQuantums and
// statefulReservedQuantums are consensus-state snapshots supplied by the
// keeper; this ledger never reads or persists consensus state itself.
func (l *SpotCandidateLedger) UpsertWithinAvailable(
	reservation SpotCandidateReservation,
	totalQuantums *big.Int,
	statefulReservedQuantums *big.Int,
) error {
	if totalQuantums == nil || statefulReservedQuantums == nil ||
		totalQuantums.Sign() < 0 || statefulReservedQuantums.Sign() < 0 {
		return ErrInvalidSpotBalance
	}

	candidate := l.Clone()
	if err := candidate.Upsert(reservation); err != nil {
		return err
	}
	available := new(big.Int).Sub(new(big.Int).Set(totalQuantums), statefulReservedQuantums)
	reserved := candidate.GetReserved(reservation.OrderId.SubaccountId, reservation.OutgoingAssetId)
	if available.Sign() < 0 || reserved.Cmp(available) > 0 {
		return ErrInsufficientSpotBalance
	}

	*l = *candidate
	return nil
}

func (l *SpotCandidateLedger) Remove(orderId types.OrderId) bool {
	reservation, exists := l.reservations[orderId]
	if !exists {
		return false
	}
	delete(l.reservations, orderId)
	l.subtract(reservation)
	l.bytes -= uint64(len(reservation.TxBytes))
	return true
}

func (l *SpotCandidateLedger) MarkPendingConfirmation(orderId types.OrderId) bool {
	reservation, exists := l.reservations[orderId]
	if !exists {
		return false
	}
	reservation.PendingConfirmation = true
	l.reservations[orderId] = reservation
	return true
}

func (l *SpotCandidateLedger) Get(orderId types.OrderId) (SpotCandidateReservation, bool) {
	reservation, exists := l.reservations[orderId]
	if !exists {
		return SpotCandidateReservation{}, false
	}
	return cloneSpotCandidateReservation(reservation), true
}

func (l *SpotCandidateLedger) GetReserved(subaccountId satypes.SubaccountId, assetId uint32) *big.Int {
	reserved, exists := l.reserved[spotCandidateBalanceKey{SubaccountId: subaccountId, AssetId: assetId}]
	if !exists {
		return new(big.Int)
	}
	return new(big.Int).Set(reserved)
}

func (l *SpotCandidateLedger) OrderCount() uint64     { return uint64(len(l.reservations)) }
func (l *SpotCandidateLedger) CandidateBytes() uint64 { return l.bytes }

// Reset discards all candidate and pending-confirmation state while preserving
// the node's configured resource limits. PrepareCheckState uses this before
// replay so reservations are rebuilt exclusively from final chain state.
func (l *SpotCandidateLedger) Reset() {
	l.reservations = make(map[types.OrderId]SpotCandidateReservation)
	l.reserved = make(map[spotCandidateBalanceKey]*big.Int)
	l.bytes = 0
}

func (l *SpotCandidateLedger) add(reservation SpotCandidateReservation) {
	key := spotCandidateBalanceKey{SubaccountId: reservation.OrderId.SubaccountId, AssetId: reservation.OutgoingAssetId}
	reserved, exists := l.reserved[key]
	if !exists {
		reserved = new(big.Int)
		l.reserved[key] = reserved
	}
	reserved.Add(reserved, reservation.ReservedQuantums)
}

func (l *SpotCandidateLedger) subtract(reservation SpotCandidateReservation) {
	key := spotCandidateBalanceKey{SubaccountId: reservation.OrderId.SubaccountId, AssetId: reservation.OutgoingAssetId}
	reserved := l.reserved[key]
	reserved.Sub(reserved, reservation.ReservedQuantums)
	if reserved.Sign() == 0 {
		delete(l.reserved, key)
	}
}

func cloneSpotCandidateReservation(reservation SpotCandidateReservation) SpotCandidateReservation {
	clone := reservation
	clone.ReservedQuantums = new(big.Int).Set(reservation.ReservedQuantums)
	clone.TxBytes = append([]byte(nil), reservation.TxBytes...)
	return clone
}
