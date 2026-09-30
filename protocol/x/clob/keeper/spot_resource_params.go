package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

func (k Keeper) GetSpotResourceParams(ctx sdk.Context) (params types.SpotResourceParams) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get([]byte(types.SpotResourceParamsKey))
	k.cdc.MustUnmarshal(b, &params)
	return params
}

func (k Keeper) SetSpotResourceParams(ctx sdk.Context, params types.SpotResourceParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.SpotResourceParamsKey), k.cdc.MustMarshal(&params))
	return nil
}
