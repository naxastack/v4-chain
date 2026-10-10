package types

import (
	"fmt"
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		BlockRateLimitConfig:  BlockRateLimitConfiguration{},
		ClobPairs:             []ClobPair{},
		EquityTierLimitConfig: EquityTierLimitConfiguration{},
		LiquidationsConfig:    LiquidationsConfig_Default,
		SpotResourceParams:    DefaultSpotResourceParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Check for duplicated id in clobPair
	clobPairIdMap := make(map[uint32]struct{})
	spotClobPairs := make(map[uint32]SpotClobMetadata)

	for _, clobPair := range gs.ClobPairs {
		if _, ok := clobPairIdMap[clobPair.Id]; ok {
			return fmt.Errorf("duplicated id for clobPair")
		}
		clobPairIdMap[clobPair.Id] = struct{}{}
		if metadata := clobPair.GetSpotClobMetadata(); metadata != nil {
			spotClobPairs[clobPair.Id] = *metadata
		}
	}

	reservationOrderIds := make(map[string]struct{}, len(gs.StatefulSpotReservations))
	for _, reservation := range gs.StatefulSpotReservations {
		orderIdKey := string(reservation.OrderId.ToStateKey())
		if _, exists := reservationOrderIds[orderIdKey]; exists {
			return fmt.Errorf("duplicated stateful spot reservation order id")
		}
		reservationOrderIds[orderIdKey] = struct{}{}

		if !reservation.OrderId.IsLongTermOrder() {
			return fmt.Errorf("stateful spot reservation must reference a long-term order")
		}
		if reservation.ReservedQuantums.IsNil() || reservation.ReservedQuantums.BigInt().Sign() <= 0 {
			return fmt.Errorf("stateful spot reservation must reserve positive quantums")
		}
		if reservation.RemainingBaseQuantums == 0 {
			return fmt.Errorf("stateful spot reservation must have positive remaining base quantums")
		}
		metadata, exists := spotClobPairs[reservation.OrderId.ClobPairId]
		if !exists {
			return fmt.Errorf("stateful spot reservation must reference a spot CLOB pair")
		}
		if reservation.OutgoingAssetId != metadata.BaseAssetId &&
			reservation.OutgoingAssetId != metadata.QuoteAssetId {
			return fmt.Errorf("stateful spot reservation outgoing asset must belong to its CLOB pair")
		}
	}

	lifecyclePairIds := make(map[uint32]struct{}, len(gs.SpotPairLifecycles))
	for _, lifecycle := range gs.SpotPairLifecycles {
		if _, exists := lifecyclePairIds[lifecycle.ClobPairId]; exists {
			return fmt.Errorf("duplicated spot pair lifecycle")
		}
		lifecyclePairIds[lifecycle.ClobPairId] = struct{}{}
		if _, exists := clobPairIdMap[lifecycle.ClobPairId]; !exists {
			return fmt.Errorf("spot pair lifecycle references an unknown clob pair")
		}
		if _, exists := spotClobPairs[lifecycle.ClobPairId]; !exists {
			return fmt.Errorf("spot pair lifecycle must reference a spot CLOB pair")
		}
		if lifecycle.PauseSource > SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL {
			return fmt.Errorf("invalid spot pair pause source")
		}
	}

	delistPairIds := make(map[uint32]struct{}, len(gs.SpotDelistCursors))
	for _, cursor := range gs.SpotDelistCursors {
		if _, exists := delistPairIds[cursor.ClobPairId]; exists {
			return fmt.Errorf("duplicated spot delist cursor")
		}
		delistPairIds[cursor.ClobPairId] = struct{}{}
		if _, exists := clobPairIdMap[cursor.ClobPairId]; !exists {
			return fmt.Errorf("spot delist cursor references an unknown clob pair")
		}
		if _, exists := spotClobPairs[cursor.ClobPairId]; !exists {
			return fmt.Errorf("spot delist cursor must reference a spot CLOB pair")
		}
		if err := ValidateSpotDelistCursor(cursor); err != nil {
			return err
		}
	}

	if err := gs.BlockRateLimitConfig.Validate(); err != nil {
		return err
	}

	if err := gs.EquityTierLimitConfig.Validate(); err != nil {
		return err
	}

	if err := gs.LiquidationsConfig.Validate(); err != nil {
		return err
	}

	if err := gs.SpotResourceParams.Validate(); err != nil {
		return err
	}

	return nil
}

func ValidateSpotDelistCursor(cursor SpotDelistCursor) error {
	if cursor.NextOrderId == (OrderId{}) {
		return nil
	}
	if err := cursor.NextOrderId.Validate(); err != nil {
		return fmt.Errorf("spot delist cursor references an illegal order id: %w", err)
	}
	if !cursor.NextOrderId.IsLongTermOrder() || cursor.NextOrderId.ClobPairId != cursor.ClobPairId {
		return fmt.Errorf("spot delist cursor references an illegal order id")
	}
	return nil
}
