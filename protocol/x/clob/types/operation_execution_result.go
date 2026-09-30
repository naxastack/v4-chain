package types

// OperationExecutionStatus describes whether an internal operation changed state.
type OperationExecutionStatus uint8

const (
	OperationExecutionStatusApplied OperationExecutionStatus = iota
	OperationExecutionStatusSkipped
)

// SpotOperationSkipReason is a stable reason for deterministically skipping a
// recoverable spot match during proposal processing.
type SpotOperationSkipReason string

const (
	SpotOperationSkipReasonInsufficientBalance       SpotOperationSkipReason = "INSUFFICIENT_BALANCE"
	SpotOperationSkipReasonOrderEpochMismatch        SpotOperationSkipReason = "ORDER_EPOCH_MISMATCH"
	SpotOperationSkipReasonOrderLifecycleInvalid     SpotOperationSkipReason = "ORDER_LIFECYCLE_INVALID"
	SpotOperationSkipReasonMarketNotTradable         SpotOperationSkipReason = "MARKET_NOT_TRADABLE"
	SpotOperationSkipReasonAssetNotTradable          SpotOperationSkipReason = "ASSET_NOT_TRADABLE"
	SpotOperationSkipReasonFeeCapExceeded            SpotOperationSkipReason = "FEE_CAP_EXCEEDED"
	SpotOperationSkipReasonQuoteNotPositive          SpotOperationSkipReason = "QUOTE_NOT_POSITIVE"
	SpotOperationSkipReasonSellerNetQuoteNotPositive SpotOperationSkipReason = "SELLER_NET_QUOTE_NOT_POSITIVE"
)

// OperationExecutionResult records the deterministic outcome of one internal
// operation. SkipReason is empty for applied operations.
type OperationExecutionResult struct {
	Operation  InternalOperation
	Status     OperationExecutionStatus
	SkipReason SpotOperationSkipReason
}
