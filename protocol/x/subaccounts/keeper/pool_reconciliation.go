package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetCollateralPoolAssetQuantums returns the aggregate subaccount ledger
// balance for an asset, grouped by collateral pool address.
func (k Keeper) GetCollateralPoolAssetQuantums(
	ctx sdk.Context,
	assetID uint32,
) (map[string]*big.Int, error) {
	balances := make(map[string]*big.Int)
	for _, subaccount := range k.GetAllSubaccount(ctx) {
		poolAddress, err := k.getCollateralPoolForSubaccount(ctx, subaccount)
		if err != nil {
			return nil, err
		}

		for _, position := range subaccount.AssetPositions {
			if position.AssetId != assetID {
				continue
			}
			key := poolAddress.String()
			if _, exists := balances[key]; !exists {
				balances[key] = new(big.Int)
			}
			balances[key].Add(balances[key], position.GetBigQuantums())
			break
		}
	}
	return balances, nil
}
