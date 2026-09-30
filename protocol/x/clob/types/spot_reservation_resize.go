package types

import "math/big"

// CalculateSpotOrderReservationForRemaining recalculates a spot reservation at the signed limit price.
func CalculateSpotOrderReservationForRemaining(
	order Order,
	pair ClobPair,
	remainingBaseQuantums uint64,
) (uint32, *big.Int, error) {
	if remainingBaseQuantums == 0 {
		metadata := pair.GetSpotClobMetadata()
		if metadata == nil {
			return 0, nil, ErrInvalidClobPairParameter
		}
		if order.IsBuy() {
			return metadata.QuoteAssetId, new(big.Int), nil
		}
		return metadata.BaseAssetId, new(big.Int), nil
	}
	remainingOrder := order
	remainingOrder.Quantums = remainingBaseQuantums
	return CalculateSpotOrderReservation(remainingOrder, pair)
}
