package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
)

func (k Keeper) GetSpotFeeParams(ctx sdk.Context) (params types.SpotFeeParams) {
	store := ctx.KVStore(k.storeKey)
	b := store.Get([]byte(types.SpotFeeParamsKey))
	k.cdc.MustUnmarshal(b, &params)
	return params
}

func (k Keeper) SetSpotFeeParams(ctx sdk.Context, params types.SpotFeeParams) error {
	if err := params.Validate(); err != nil {
		return err
	}
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.SpotFeeParamsKey), k.cdc.MustMarshal(&params))
	return nil
}
