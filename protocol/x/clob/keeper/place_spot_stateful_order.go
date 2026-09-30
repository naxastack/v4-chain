package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// placeSpotStatefulOrder commits the order and all reservation state atomically.
func (k Keeper) placeSpotStatefulOrder(ctx sdk.Context, order types.Order) error {
	if k.HasStatefulSpotReservation(ctx, order.OrderId) {
		return types.ErrStatefulOrderAlreadyExists
	}
	params := k.GetSpotResourceParams(ctx)
	count := k.getSpotStatefulOrderCount(ctx, order.OrderId.SubaccountId)
	if count >= params.MaxSpotStatefulOrdersPerSubaccount {
		return errorsmod.Wrap(types.ErrInvalidPlaceOrder, "spot stateful order limit exceeded")
	}
	pair, found := k.GetClobPair(ctx, order.GetClobPairId())
	if !found {
		return types.ErrInvalidClob
	}
	metadata := pair.GetSpotClobMetadata()
	if metadata == nil {
		return errorsmod.Wrap(types.ErrInvalidPlaceOrder, "spot reservation requires a spot CLOB pair")
	}
	if err := k.assetsKeeper.ValidateAssetForSpotTrading(ctx, metadata.BaseAssetId); err != nil {
		return err
	}
	if err := k.assetsKeeper.ValidateAssetForSpotTrading(ctx, metadata.QuoteAssetId); err != nil {
		return err
	}
	assetId, required, err := types.CalculateSpotOrderReservation(order, pair)
	if err != nil {
		return err
	}

	cacheCtx, write := ctx.CacheContext()
	if err := k.subaccountsKeeper.ReserveStatefulSpotQuantums(
		cacheCtx,
		order.OrderId.SubaccountId,
		assetId,
		required,
	); err != nil {
		return err
	}
	k.SetStatefulSpotReservation(cacheCtx, types.StatefulSpotReservation{
		OrderId:               order.OrderId,
		OutgoingAssetId:       assetId,
		ReservedQuantums:      dtypes.NewIntFromBigInt(required),
		RemainingBaseQuantums: order.Quantums,
	})
	k.setSpotStatefulOrderCount(cacheCtx, order.OrderId.SubaccountId, count+1)
	k.SetLongTermOrderPlacement(cacheCtx, order, lib.MustConvertIntegerToUint32(ctx.BlockHeight()))
	k.AddStatefulOrderIdExpiration(
		cacheCtx,
		order.MustGetUnixGoodTilBlockTime(),
		order.GetOrderId(),
	)
	write()
	return nil
}
