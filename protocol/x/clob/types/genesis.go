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
