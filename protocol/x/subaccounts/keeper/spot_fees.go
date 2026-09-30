package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// TransferSpotFees transfers positive spot trading fees from the default
// subaccounts collateral pool to the fee collector.
func (k Keeper) TransferSpotFees(
	ctx sdk.Context,
	assetId uint32,
	quantums *big.Int,
) error {
	if quantums == nil || quantums.Sign() <= 0 {
		return types.ErrAssetTransferQuantumsNotPositive
	}
	return k.TransferFees(
		ctx,
		assetId,
		types.ModuleAddress,
		authtypes.NewModuleAddress(authtypes.FeeCollectorName),
		quantums,
	)
}
