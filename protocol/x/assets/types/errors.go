package types

// DONTCOVER

import errorsmod "cosmossdk.io/errors"

// x/assets module sentinel errors
var (
	ErrAssetDoesNotExist            = errorsmod.Register(ModuleName, 1, "Asset does not exist")
	ErrNoAssetWithDenom             = errorsmod.Register(ModuleName, 3, "No asset found associated with given denom")
	ErrAssetDenomAlreadyExists      = errorsmod.Register(ModuleName, 4, "Existing asset found with the same denom")
	ErrAssetIdAlreadyExists         = errorsmod.Register(ModuleName, 5, "Existing asset found with the same asset id")
	ErrGapFoundInAssetId            = errorsmod.Register(ModuleName, 6, "Found gap in asset Id")
	ErrUsdcMustBeAssetZero          = errorsmod.Register(ModuleName, 7, "USDC must be asset 0")
	ErrNoAssetInGenesis             = errorsmod.Register(ModuleName, 8, "No asset found in genesis state")
	ErrInvalidMarketId              = errorsmod.Register(ModuleName, 9, "Found market id for asset without market")
	ErrInvalidAssetAtomicResolution = errorsmod.Register(ModuleName, 10, "Invalid asset atomic resolution")
	ErrInvalidDenomExponent         = errorsmod.Register(ModuleName, 11, "Invalid denom exponent")
	ErrAssetAlreadyExists           = errorsmod.Register(ModuleName, 12, "Asset already exists")
	ErrUnexpectedUsdcDenomExponent  = errorsmod.Register(ModuleName, 13, "USDC denom exponent is unexpected")
	ErrInvalidAssetPolicyStatus     = errorsmod.Register(ModuleName, 14, "Invalid asset policy status")
	ErrInvalidAssetPolicyTransition = errorsmod.Register(ModuleName, 15, "Invalid asset policy transition")
	ErrAssetPolicyAssetIdMismatch   = errorsmod.Register(ModuleName, 16, "Asset policy asset id mismatch")
	ErrAssetExitMustRemainEnabled   = errorsmod.Register(ModuleName, 17, "Asset withdrawal permission must remain enabled")
	ErrAssetPolicyDoesNotExist      = errorsmod.Register(ModuleName, 18, "Asset policy does not exist")
	ErrAssetPolicyAlreadyExists     = errorsmod.Register(ModuleName, 19, "Asset policy already exists")
	ErrInvalidAssetDenom            = errorsmod.Register(ModuleName, 20, "Invalid asset denom")
	ErrAssetDepositsDisabled        = errorsmod.Register(ModuleName, 21, "Asset deposits are disabled")
	ErrAssetWithdrawalsDisabled     = errorsmod.Register(ModuleName, 22, "Asset withdrawals are disabled")
	ErrAssetSpotTradingDisabled     = errorsmod.Register(ModuleName, 23, "Asset spot trading is disabled")
	ErrAssetPerpetualDisabled       = errorsmod.Register(ModuleName, 24, "Asset perpetual use is disabled")
	ErrAssetDenomMismatch           = errorsmod.Register(ModuleName, 25, "Asset denom does not match asset id")

	// Errors for Not Implemented
	ErrNotImplementedMulticollateral = errorsmod.Register(ModuleName, 401, "Not Implemented: Multi-Collateral")
	ErrNotImplementedMargin          = errorsmod.Register(ModuleName, 402, "Not Implemented: Margin-Trading of Assets")
)
