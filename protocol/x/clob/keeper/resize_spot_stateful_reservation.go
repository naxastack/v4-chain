package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// ResizeStatefulSpotReservation reduces an active spot order reservation atomically.
func (k Keeper) ResizeStatefulSpotReservation(
	ctx sdk.Context,
	orderId types.OrderId,
	remainingBaseQuantums uint64,
) error {
	reservation, found := k.GetStatefulSpotReservation(ctx, orderId)
	if !found {
		return types.ErrStatefulOrderDoesNotExist
	}
	if !k.HasStatefulSpotOrderMarketIndex(ctx, orderId) {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation market index does not exist")
	}
	placement, found := k.GetLongTermOrderPlacement(ctx, orderId)
	if !found {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation exists without an order")
	}
	if remainingBaseQuantums > reservation.RemainingBaseQuantums {
		return errorsmod.Wrap(types.ErrInvalidPlaceOrder, "spot reservation remaining quantity cannot increase")
	}
	if remainingBaseQuantums == reservation.RemainingBaseQuantums {
		return nil
	}
	if remainingBaseQuantums == 0 {
		return k.ReleaseStatefulSpotOrder(ctx, orderId)
	}
	pair, found := k.GetClobPair(ctx, types.ClobPairId(orderId.GetClobPairId()))
	if !found || pair.GetSpotClobMetadata() == nil {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation references a non-spot CLOB pair")
	}
	assetId, required, err := types.CalculateSpotOrderReservationForRemaining(
		placement.Order,
		pair,
		remainingBaseQuantums,
	)
	if err != nil {
		return err
	}
	if assetId != reservation.OutgoingAssetId {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "recalculated reservation asset changed")
	}
	oldRequired := reservation.ReservedQuantums.BigInt()
	if oldRequired == nil || oldRequired.Sign() <= 0 || required.Cmp(oldRequired) > 0 {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "recalculated reservation cannot increase")
	}
	release := new(big.Int).Sub(new(big.Int).Set(oldRequired), required)

	cacheCtx, write := ctx.CacheContext()
	if release.Sign() > 0 {
		if err := k.subaccountsKeeper.ReleaseStatefulSpotQuantums(
			cacheCtx,
			orderId.SubaccountId,
			assetId,
			release,
		); err != nil {
			return err
		}
	}
	reservation.ReservedQuantums = dtypes.NewIntFromBigInt(required)
	reservation.RemainingBaseQuantums = remainingBaseQuantums
	k.SetStatefulSpotReservation(cacheCtx, reservation)
	write()
	return nil
}
