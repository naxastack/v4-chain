package types

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// ValidateSpotLongTermOrder validates the product-specific fields of a spot long-term order.
func ValidateSpotLongTermOrder(order Order, currentFeePpm uint32) error {
	if !order.OrderId.IsLongTermOrder() {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot stateful orders must use long-term order flags")
	}
	if order.TimeInForce != Order_TIME_IN_FORCE_UNSPECIFIED {
		return errorsmod.Wrap(ErrUnexpectedTimeInForce, "spot long-term orders require unspecified time in force")
	}
	if order.Quantums == 0 {
		return errorsmod.Wrap(ErrInvalidOrderQuantums, "spot order quantums cannot be zero")
	}
	if order.Subticks == 0 {
		return errorsmod.Wrap(ErrInvalidOrderSubticks, "spot order subticks cannot be zero")
	}
	if order.ReduceOnly || order.ConditionType != Order_CONDITION_TYPE_UNSPECIFIED ||
		order.ConditionalOrderTriggerSubticks != 0 || order.TwapParameters != nil {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot long-term order contains unsupported execution fields")
	}
	if order.BuilderCodeParameters != nil || order.OrderRouterAddress != "" {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot long-term order contains unsupported routing or builder fields")
	}
	if order.MaxTradingFeePpm == 0 || order.MaxTradingFeePpm < currentFeePpm {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot trading fee exceeds the signed fee limit")
	}
	if order.MaxTradingFeePpm > uint32(lib.OneMillion) {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot trading fee limit cannot exceed one million ppm")
	}
	if order.OrderEpoch != 0 {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "spot long-term order epoch must be zero")
	}
	return nil
}

// ValidatePerpetualOrderSpotFields rejects spot-only fields on perpetual orders.
func ValidatePerpetualOrderSpotFields(order Order) error {
	if order.MaxTradingFeePpm != 0 || order.OrderEpoch != 0 {
		return errorsmod.Wrap(ErrInvalidPlaceOrder, "perpetual orders cannot set spot-only fields")
	}
	return nil
}

// CalculateSpotOrderReservation returns the outgoing asset and maximum reservation for a spot order.
func CalculateSpotOrderReservation(order Order, pair ClobPair) (uint32, *big.Int, error) {
	metadata := pair.GetSpotClobMetadata()
	if metadata == nil {
		return 0, nil, errorsmod.Wrap(ErrInvalidClobPairParameter, "spot metadata is required")
	}
	if pair.StepBaseQuantums == 0 || pair.SubticksPerTick == 0 || order.Quantums == 0 || order.Subticks == 0 {
		return 0, nil, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot reservation inputs must be non-zero")
	}
	if order.Quantums%pair.StepBaseQuantums != 0 || order.Subticks%uint64(pair.SubticksPerTick) != 0 {
		return 0, nil, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot order does not satisfy pair step or tick size")
	}
	quote := FillAmountToQuoteQuantums(
		Subticks(order.Subticks),
		satypes.BaseQuantums(order.Quantums),
		pair.QuantumConversionExponent,
	)
	if quote.Sign() <= 0 {
		return 0, nil, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot order quote amount must be positive")
	}
	fee := new(big.Int).Mul(new(big.Int).Set(quote), new(big.Int).SetUint64(uint64(order.MaxTradingFeePpm)))
	fee = lib.BigDivCeil(fee, big.NewInt(int64(lib.OneMillion)))
	if !order.IsBuy() {
		if new(big.Int).Sub(new(big.Int).Set(quote), fee).Sign() <= 0 {
			return 0, nil, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot seller net quote amount must be positive")
		}
		return metadata.BaseAssetId, new(big.Int).SetUint64(order.Quantums), nil
	}
	fillCount := new(big.Int).SetUint64(order.Quantums / pair.StepBaseQuantums)
	roundingAllowance := new(big.Int).Sub(fillCount, big.NewInt(1))
	if roundingAllowance.Sign() < 0 {
		roundingAllowance.SetInt64(0)
	}
	required := new(big.Int).Add(new(big.Int).Set(quote), fee)
	required.Add(required, roundingAllowance)
	return metadata.QuoteAssetId, required, nil
}
