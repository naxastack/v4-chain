package types

import "math/big"

// SpotCandidateState is the consensus-state snapshot used to validate one
// node-local spot short-term reservation.
type SpotCandidateState struct {
	ClobPair                 ClobPair
	OutgoingAssetId          uint32
	TotalQuantums            *big.Int
	StatefulReservedQuantums *big.Int
	CurrentOrderEpoch        uint64
}
