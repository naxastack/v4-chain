package keeper

import (
	"encoding/binary"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// GetStatefulSpotReservation returns the reservation for an active spot order.
func (k Keeper) GetStatefulSpotReservation(
	ctx sdk.Context,
	orderId types.OrderId,
) (types.StatefulSpotReservation, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.StatefulSpotReservationKeyPrefix))
	value := store.Get(orderId.ToStateKey())
	if value == nil {
		return types.StatefulSpotReservation{}, false
	}
	var reservation types.StatefulSpotReservation
	k.cdc.MustUnmarshal(value, &reservation)
	return reservation, true
}

// SetStatefulSpotReservation stores one reservation and its market index.
func (k Keeper) SetStatefulSpotReservation(
	ctx sdk.Context,
	reservation types.StatefulSpotReservation,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.StatefulSpotReservationKeyPrefix))
	store.Set(reservation.OrderId.ToStateKey(), k.cdc.MustMarshal(&reservation))

	indexStore := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		spotStatefulOrdersByClobPairPrefix(reservation.OrderId.ClobPairId),
	)
	indexStore.Set(reservation.OrderId.ToStateKey(), reservation.OrderId.ToStateKey())
}

// ImportStatefulSpotReservation restores one reservation and its account order count.
func (k Keeper) ImportStatefulSpotReservation(
	ctx sdk.Context,
	reservation types.StatefulSpotReservation,
) {
	k.SetStatefulSpotReservation(ctx, reservation)
	id := reservation.OrderId.SubaccountId
	k.setSpotStatefulOrderCount(ctx, id, k.getSpotStatefulOrderCount(ctx, id)+1)
}

// DeleteStatefulSpotReservation removes one reservation and its market index.
func (k Keeper) DeleteStatefulSpotReservation(
	ctx sdk.Context,
	reservation types.StatefulSpotReservation,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.StatefulSpotReservationKeyPrefix))
	store.Delete(reservation.OrderId.ToStateKey())
	indexStore := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		spotStatefulOrdersByClobPairPrefix(reservation.OrderId.ClobPairId),
	)
	indexStore.Delete(reservation.OrderId.ToStateKey())
}

// GetAllStatefulSpotReservations returns all reservations in deterministic state-key order.
func (k Keeper) GetAllStatefulSpotReservations(ctx sdk.Context) []types.StatefulSpotReservation {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.StatefulSpotReservationKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	reservations := make([]types.StatefulSpotReservation, 0)
	for ; iterator.Valid(); iterator.Next() {
		var reservation types.StatefulSpotReservation
		k.cdc.MustUnmarshal(iterator.Value(), &reservation)
		reservations = append(reservations, reservation)
	}
	return reservations
}

// HasStatefulSpotReservation reports whether an order already owns a reservation.
func (k Keeper) HasStatefulSpotReservation(ctx sdk.Context, orderId types.OrderId) bool {
	_, found := k.GetStatefulSpotReservation(ctx, orderId)
	return found
}

// HasStatefulSpotOrderMarketIndex reports whether the reservation market index exists.
func (k Keeper) HasStatefulSpotOrderMarketIndex(ctx sdk.Context, orderId types.OrderId) bool {
	indexStore := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		spotStatefulOrdersByClobPairPrefix(orderId.ClobPairId),
	)
	return indexStore.Has(orderId.ToStateKey())
}

func spotStatefulOrdersByClobPairPrefix(clobPairId uint32) []byte {
	key := append([]byte(types.SpotStatefulOrdersByClobPairKeyPrefix), lib.Uint32ToKey(clobPairId)...)
	return key
}
func (k Keeper) getSpotStatefulOrderCount(ctx sdk.Context, id satypes.SubaccountId) uint32 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotStatefulOrderCountKeyPrefix))
	value := store.Get((&id).ToStateKey())
	if len(value) != 4 {
		return 0
	}
	return binary.BigEndian.Uint32(value)
}

func (k Keeper) setSpotStatefulOrderCount(
	ctx sdk.Context,
	id satypes.SubaccountId,
	count uint32,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotStatefulOrderCountKeyPrefix))
	key := (&id).ToStateKey()
	if count == 0 {
		store.Delete(key)
		return
	}
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, count)
	store.Set(key, value)
}
