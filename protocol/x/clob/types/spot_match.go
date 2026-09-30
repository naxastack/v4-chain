package types

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
)

// SpotMatchSettlement contains the asset deltas and fees for a spot match.
// All deltas are signed from the perspective of the indicated participant.
type SpotMatchSettlement struct {
	BaseAssetId            uint32
	QuoteAssetId           uint32
	BaseQuantums           *big.Int
	QuoteQuantums          *big.Int
	BuyerFeeQuoteQuantums  *big.Int
	SellerFeeQuoteQuantums *big.Int
	BuyerBaseDelta         *big.Int
	BuyerQuoteDelta        *big.Int
	SellerBaseDelta        *big.Int
	SellerQuoteDelta       *big.Int
	MakerIsBuyer           bool
}

// CalculateSpotMatchSettlement calculates spot asset deltas at the maker price.
// It performs no state reads or writes.
func CalculateSpotMatchSettlement(match MatchWithOrders, pair ClobPair, tradingFeePpm uint32) (SpotMatchSettlement, error) {
	result := SpotMatchSettlement{}
	if err := match.Validate(); err != nil {
		return result, errorsmod.Wrap(ErrInvalidPlaceOrder, err.Error())
	}
	if match.MakerOrder.GetClobPairId().ToUint32() != pair.Id {
		return result, errorsmod.Wrap(ErrInvalidClob, "spot match and pair ids do not match")
	}
	metadata := pair.GetSpotClobMetadata()
	if metadata == nil {
		return result, errorsmod.Wrap(ErrInvalidClobPairParameter, "spot metadata is required")
	}
	if pair.StepBaseQuantums == 0 || match.FillAmount.ToUint64()%pair.StepBaseQuantums != 0 {
		return result, ErrFillAmountNotDivisibleByStepSize
	}
	if tradingFeePpm == 0 || tradingFeePpm > uint32(lib.OneMillion) {
		return result, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot trading fee ppm must be between 1 and one million")
	}
	if match.MakerOrder.IsLiquidation() || match.TakerOrder.IsLiquidation() {
		return result, errorsmod.Wrap(ErrInvalidPlaceOrder, "spot matches cannot contain liquidation orders")
	}
	makerOrder := match.MakerOrder.MustGetOrder()
	takerOrder := match.TakerOrder.MustGetOrder()
	if makerOrder.MaxTradingFeePpm < tradingFeePpm || takerOrder.MaxTradingFeePpm < tradingFeePpm {
		return result, ErrSpotFeeCapExceeded
	}

	baseQuantums := new(big.Int).SetUint64(match.FillAmount.ToUint64())
	quoteQuantums := FillAmountToQuoteQuantums(match.MakerOrder.GetOrderSubticks(), match.FillAmount, pair.QuantumConversionExponent)
	if quoteQuantums.Sign() <= 0 {
		return result, ErrSpotQuoteNotPositive
	}
	feeNumerator := new(big.Int).Mul(new(big.Int).Set(quoteQuantums), new(big.Int).SetUint64(uint64(tradingFeePpm)))
	fee := lib.BigDivCeil(feeNumerator, big.NewInt(int64(lib.OneMillion)))
	sellerQuoteDelta := new(big.Int).Sub(new(big.Int).Set(quoteQuantums), fee)
	if sellerQuoteDelta.Sign() <= 0 {
		return result, ErrSpotSellerNetQuoteNotPositive
	}

	result = SpotMatchSettlement{
		BaseAssetId: metadata.BaseAssetId, QuoteAssetId: metadata.QuoteAssetId,
		BaseQuantums: new(big.Int).Set(baseQuantums), QuoteQuantums: new(big.Int).Set(quoteQuantums),
		BuyerFeeQuoteQuantums: new(big.Int).Set(fee), SellerFeeQuoteQuantums: new(big.Int).Set(fee),
		BuyerBaseDelta:   new(big.Int).Set(baseQuantums),
		BuyerQuoteDelta:  new(big.Int).Neg(new(big.Int).Add(new(big.Int).Set(quoteQuantums), fee)),
		SellerBaseDelta:  new(big.Int).Neg(new(big.Int).Set(baseQuantums)),
		SellerQuoteDelta: sellerQuoteDelta, MakerIsBuyer: match.MakerOrder.IsBuy(),
	}
	return result, nil
}
