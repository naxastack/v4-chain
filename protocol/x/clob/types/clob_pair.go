package types

import (
	errorsmod "cosmossdk.io/errors"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// SupportedClobPairStatusTransitions has keys corresponding to currently-supported
// ClobPair_Status types with values equal to the set of ClobPair_Status types that
// may be transitioned to from this state. Note the keys of this map may be
// a subset of the types defined in the proto for ClobPair_Status.
var SupportedClobPairStatusTransitions = map[ClobPair_Status]map[ClobPair_Status]struct{}{
	ClobPair_STATUS_ACTIVE: {
		ClobPair_STATUS_FINAL_SETTLEMENT: struct{}{},
	},
	ClobPair_STATUS_INITIALIZING: {
		ClobPair_STATUS_ACTIVE:           struct{}{},
		ClobPair_STATUS_FINAL_SETTLEMENT: struct{}{},
	},
	ClobPair_STATUS_FINAL_SETTLEMENT: {
		ClobPair_STATUS_INITIALIZING: struct{}{},
	},
}

// SupportedSpotClobPairStatusTransitions is separate from the perpetual
// transition table. Spot final settlement is terminal and CANCEL_ONLY is a
// recoverable trading pause.
var SupportedSpotClobPairStatusTransitions = map[ClobPair_Status]map[ClobPair_Status]struct{}{
	ClobPair_STATUS_ACTIVE: {
		ClobPair_STATUS_CANCEL_ONLY:      {},
		ClobPair_STATUS_FINAL_SETTLEMENT: {},
	},
	ClobPair_STATUS_CANCEL_ONLY: {
		ClobPair_STATUS_ACTIVE:           {},
		ClobPair_STATUS_FINAL_SETTLEMENT: {},
	},
	ClobPair_STATUS_FINAL_SETTLEMENT: {},
}

func IsSupportedSpotClobPairStatus(status ClobPair_Status) bool {
	_, exists := SupportedSpotClobPairStatusTransitions[status]
	return exists
}

// IsSupportedSpotClobPairStatusTransition validates the Spot-only lifecycle.
// Repeating a status is idempotent; FINAL_SETTLEMENT cannot transition out.
func IsSupportedSpotClobPairStatusTransition(from, to ClobPair_Status) bool {
	if !IsSupportedSpotClobPairStatus(from) || !IsSupportedSpotClobPairStatus(to) {
		return false
	}
	if from == to {
		return true
	}
	_, exists := SupportedSpotClobPairStatusTransitions[from][to]
	return exists
}

// IsSupportedClobPairStatus returns true if the provided ClobPair_Status is in the list
// of currently supported ClobPair_Status types. Else, returns false.
func IsSupportedClobPairStatus(clobPairStatus ClobPair_Status) bool {
	_, exists := SupportedClobPairStatusTransitions[clobPairStatus]
	return exists
}

// IsSupportedClobPairStatusTransition returns true if it is considered valid to transition from
// the first provided ClobPair_Status to the second provided ClobPair_Status. Else, returns false.
// Transitions from a ClobPair_Status to itself are considered valid.
func IsSupportedClobPairStatusTransition(from ClobPair_Status, to ClobPair_Status) bool {
	if !IsSupportedClobPairStatus(from) || !IsSupportedClobPairStatus(to) {
		return false
	}

	if from == to {
		return true
	}

	_, exists := SupportedClobPairStatusTransitions[from][to]
	return exists
}

func (c *ClobPair) GetClobPairSubticksPerTick() SubticksPerTick {
	return SubticksPerTick(c.SubticksPerTick)
}

func (c *ClobPair) GetClobPairMinOrderBaseQuantums() satypes.BaseQuantums {
	return satypes.BaseQuantums(c.StepBaseQuantums)
}

// GetPerpetualId returns the `PerpetualId` for the provided `clobPair`.
func (c *ClobPair) GetPerpetualId() (uint32, error) {
	perpetualClobMetadata := c.GetPerpetualClobMetadata()
	if perpetualClobMetadata == nil {
		return 0, ErrAssetOrdersNotImplemented
	}

	return perpetualClobMetadata.PerpetualId, nil
}

// MustGetPerpetualId returns the `PerpetualId` for the provided `clobPair`.
// Will panic if `GetPerpetualId` returns an error.
func (c *ClobPair) MustGetPerpetualId() uint32 {
	id, err := c.GetPerpetualId()
	if err != nil {
		panic(err)
	}
	return id
}

// GetId returns the `ClobPairId` for the provided `clobPair`.
func (c *ClobPair) GetClobPairId() ClobPairId {
	return ClobPairId(c.Id)
}

// Stateless validation on ClobPair.
func (c *ClobPair) Validate() error {
	isSpot := false
	switch metadata := c.Metadata.(type) {
	case *ClobPair_PerpetualClobMetadata:
		if metadata.PerpetualClobMetadata == nil {
			return errorsmod.Wrap(ErrInvalidClobPairParameter, "perpetual CLOB metadata cannot be nil")
		}
	case *ClobPair_SpotClobMetadata:
		isSpot = true
		if metadata.SpotClobMetadata == nil {
			return errorsmod.Wrap(ErrInvalidClobPairParameter, "spot CLOB metadata cannot be nil")
		}
		if metadata.SpotClobMetadata.BaseAssetId == metadata.SpotClobMetadata.QuoteAssetId {
			return errorsmod.Wrap(ErrInvalidClobPairParameter, "spot base and quote assets must differ")
		}
	default:
		return errorsmod.Wrap(ErrInvalidClobPairParameter, "CLOB metadata must be specified")
	}

	isSupportedStatus := IsSupportedClobPairStatus(c.Status)
	if isSpot {
		isSupportedStatus = IsSupportedSpotClobPairStatus(c.Status)
	}
	if !isSupportedStatus {
		return errorsmod.Wrapf(
			ErrInvalidClobPairParameter,
			"CLOB pair (%+v) has unsupported status %+v",
			c,
			c.Status,
		)
	}

	if c.StepBaseQuantums <= 0 {
		return errorsmod.Wrapf(
			ErrInvalidClobPairParameter,
			"invalid ClobPair parameter: StepBaseQuantums must be > 0. Got %v",
			c.StepBaseQuantums,
		)
	}

	// Since a subtick will be calculated as (1 tick/SubticksPerTick), the denominator cannot be 0
	// and negative numbers do not make sense.
	if c.SubticksPerTick <= 0 {
		return errorsmod.Wrapf(
			ErrInvalidClobPairParameter,
			"invalid ClobPair parameter: SubticksPerTick must be > 0. Got %v",
			c.SubticksPerTick,
		)
	}

	return nil
}
