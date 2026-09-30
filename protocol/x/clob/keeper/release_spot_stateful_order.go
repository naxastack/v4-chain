package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// ReleaseStatefulSpotOrder atomically removes an active spot order and releases its reservation.
func (k Keeper) ReleaseStatefulSpotOrder(ctx sdk.Context, orderId types.OrderId) error {
	reservation, found := k.GetStatefulSpotReservation(ctx, orderId)
	if !found {
		return types.ErrStatefulOrderDoesNotExist
	}
	if !k.HasStatefulSpotOrderMarketIndex(ctx, orderId) {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation market index does not exist")
	}
	_, found = k.getOrderFromStore(ctx, orderId)
	if !found {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation exists without an order")
	}
	pair, found := k.GetClobPair(ctx, types.ClobPairId(orderId.GetClobPairId()))
	if !found || pair.GetSpotClobMetadata() == nil {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation references a non-spot CLOB pair")
	}
	metadata := pair.GetSpotClobMetadata()
	if reservation.OutgoingAssetId != metadata.BaseAssetId &&
		reservation.OutgoingAssetId != metadata.QuoteAssetId {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation asset does not belong to its CLOB pair")
	}
	count := k.getSpotStatefulOrderCount(ctx, orderId.SubaccountId)
	if count == 0 {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot active order count is zero")
	}
	amount := reservation.ReservedQuantums.BigInt()
	if amount == nil || amount.Sign() < 0 ||
		(amount.Sign() == 0 && reservation.RemainingBaseQuantums != 0) {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation amount is invalid")
	}

	cacheCtx, write := ctx.CacheContext()
	if amount.Sign() > 0 {
		if err := k.subaccountsKeeper.ReleaseStatefulSpotQuantums(
			cacheCtx,
			orderId.SubaccountId,
			reservation.OutgoingAssetId,
			amount,
		); err != nil {
			return err
		}
	}
	k.DeleteStatefulSpotReservation(cacheCtx, reservation)
	k.setSpotStatefulOrderCount(cacheCtx, orderId.SubaccountId, count-1)
	k.mustRemoveStatefulOrder(cacheCtx, orderId)
	write()
	return nil
}

func (k Keeper) mustReleaseStatefulSpotOrder(ctx sdk.Context, orderId types.OrderId) {
	if err := k.ReleaseStatefulSpotOrder(ctx, orderId); err != nil {
		panic(fmt.Sprintf("MustRemoveStatefulOrder: failed to release spot order %v: %v", orderId, err))
	}
}
