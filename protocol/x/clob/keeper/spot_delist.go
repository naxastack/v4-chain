package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// EnsureSpotDelistCursor starts order cleanup for a Spot pair entering final settlement.
func (k Keeper) EnsureSpotDelistCursor(ctx sdk.Context, clobPairId uint32) {
	if _, found := k.GetSpotDelistCursor(ctx, clobPairId); found {
		return
	}
	k.SetSpotDelistCursor(ctx, types.SpotDelistCursor{ClobPairId: clobPairId})
}

func (k Keeper) GetSpotDelistCursor(
	ctx sdk.Context,
	clobPairId uint32,
) (types.SpotDelistCursor, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotDelistCursorKeyPrefix))
	value := store.Get(lib.Uint32ToKey(clobPairId))
	if value == nil {
		return types.SpotDelistCursor{}, false
	}
	var cursor types.SpotDelistCursor
	k.cdc.MustUnmarshal(value, &cursor)
	return cursor, true
}

func (k Keeper) SetSpotDelistCursor(ctx sdk.Context, cursor types.SpotDelistCursor) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotDelistCursorKeyPrefix))
	store.Set(lib.Uint32ToKey(cursor.ClobPairId), k.cdc.MustMarshal(&cursor))
}

func (k Keeper) GetAllSpotDelistCursors(ctx sdk.Context) []types.SpotDelistCursor {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotDelistCursorKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	cursors := make([]types.SpotDelistCursor, 0)
	for ; iterator.Valid(); iterator.Next() {
		var cursor types.SpotDelistCursor
		k.cdc.MustUnmarshal(iterator.Value(), &cursor)
		cursors = append(cursors, cursor)
	}
	return cursors
}

func (k Keeper) GetSpotDelistSchedulerCursor(ctx sdk.Context) types.SpotDelistSchedulerCursor {
	value := ctx.KVStore(k.storeKey).Get([]byte(types.SpotDelistSchedulerCursorKey))
	if len(value) != 4 {
		return types.SpotDelistSchedulerCursor{}
	}
	return types.SpotDelistSchedulerCursor{NextClobPairId: lib.BytesToUint32(value)}
}

func (k Keeper) SetSpotDelistSchedulerCursor(ctx sdk.Context, cursor types.SpotDelistSchedulerCursor) {
	ctx.KVStore(k.storeKey).Set(
		[]byte(types.SpotDelistSchedulerCursorKey),
		lib.Uint32ToKey(cursor.NextClobPairId),
	)
}

// CancelSpotStatefulOrder releases one Spot long-term order. The reservation,
// index, and order placement commit together only when every step succeeds.
func (k Keeper) CancelSpotStatefulOrder(ctx sdk.Context, orderId types.OrderId) error {
	if !orderId.IsLongTermOrder() {
		return types.ErrInvalidOrderRemoval
	}
	pair, found := k.GetClobPair(ctx, types.ClobPairId(orderId.ClobPairId))
	if !found || pair.GetSpotClobMetadata() == nil {
		return types.ErrInvalidOrderRemoval
	}
	return k.ReleaseStatefulSpotOrder(ctx, orderId)
}

// AdvanceSpotDelistOrders cancels at most one long-term order per final-settlement
// pair before rotating, until the block limit is exhausted.
func (k Keeper) AdvanceSpotDelistOrders(ctx sdk.Context) []types.OrderId {
	pairIds := k.incompleteSpotDelistPairIds(ctx)
	if len(pairIds) == 0 {
		return nil
	}
	limit := k.GetSpotResourceParams(ctx).MaxSpotDelistOrdersPerBlock
	if limit == 0 {
		return nil
	}

	cursor := k.GetSpotDelistSchedulerCursor(ctx).NextClobPairId
	start := 0
	for i, pairId := range pairIds {
		if pairId >= cursor {
			start = i
			break
		}
	}

	removed := make([]types.OrderId, 0)
	advanced := uint32(0)
	idle := 0
	idx := start
	for advanced < limit && idle < len(pairIds) {
		orderId, ok := k.cancelNextSpotDelistOrder(ctx, pairIds[idx])
		if ok {
			removed = append(removed, orderId)
			advanced++
			idle = 0
			k.SetSpotDelistSchedulerCursor(ctx, types.SpotDelistSchedulerCursor{
				NextClobPairId: pairIds[(idx+1)%len(pairIds)],
			})
		} else {
			idle++
		}
		idx = (idx + 1) % len(pairIds)
	}
	return removed
}

func (k Keeper) cancelNextSpotDelistOrder(ctx sdk.Context, clobPairId uint32) (types.OrderId, bool) {
	cursor, found := k.GetSpotDelistCursor(ctx, clobPairId)
	if !found || cursor.Completed {
		return types.OrderId{}, false
	}
	orderId, exists := k.nextIndexedSpotOrder(ctx, clobPairId, cursor.NextOrderId)
	if !exists {
		cursor.Completed = true
		cursor.NextOrderId = types.OrderId{}
		k.SetSpotDelistCursor(ctx, cursor)
		return types.OrderId{}, false
	}
	if err := k.CancelSpotStatefulOrder(ctx, orderId); err != nil {
		return types.OrderId{}, false
	}
	nextOrderId, hasNext := k.nextIndexedSpotOrder(ctx, clobPairId, orderId)
	cursor.NextOrderId = nextOrderId
	cursor.Completed = !hasNext
	if !hasNext {
		cursor.NextOrderId = types.OrderId{}
	}
	k.SetSpotDelistCursor(ctx, cursor)
	return orderId, true
}

func (k Keeper) nextIndexedSpotOrder(
	ctx sdk.Context,
	clobPairId uint32,
	start types.OrderId,
) (types.OrderId, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), spotStatefulOrdersByClobPairPrefix(clobPairId))
	var startKey []byte
	if start != (types.OrderId{}) {
		startKey = start.ToStateKey()
	}
	iterator := store.Iterator(startKey, nil)
	defer iterator.Close()
	if !iterator.Valid() {
		return types.OrderId{}, false
	}
	var orderId types.OrderId
	if err := orderId.Unmarshal(iterator.Key()); err != nil {
		panic(err)
	}
	return orderId, true
}

func (k Keeper) incompleteSpotDelistPairIds(ctx sdk.Context) []uint32 {
	pairIds := make([]uint32, 0)
	for _, cursor := range k.GetAllSpotDelistCursors(ctx) {
		if cursor.Completed {
			continue
		}
		pair, found := k.GetClobPair(ctx, types.ClobPairId(cursor.ClobPairId))
		if !found || pair.GetSpotClobMetadata() == nil || pair.Status != types.ClobPair_STATUS_FINAL_SETTLEMENT {
			continue
		}
		pairIds = append(pairIds, cursor.ClobPairId)
	}
	return pairIds
}

func (k Keeper) ImportSpotDelistCursor(ctx sdk.Context, cursor types.SpotDelistCursor) error {
	pair, found := k.GetClobPair(ctx, types.ClobPairId(cursor.ClobPairId))
	if !found || pair.GetSpotClobMetadata() == nil {
		return types.ErrInvalidSpotDelistCursor
	}
	if err := types.ValidateSpotDelistCursor(cursor); err != nil {
		return err
	}
	k.SetSpotDelistCursor(ctx, cursor)
	return nil
}

func (k Keeper) EnsureFinalSettlementDelistCursors(ctx sdk.Context) {
	for _, pair := range k.GetAllClobPairs(ctx) {
		if pair.GetSpotClobMetadata() == nil {
			continue
		}
		state, found := k.GetSpotPairLifecycleState(ctx, pair.GetClobPairId())
		if pair.Status == types.ClobPair_STATUS_FINAL_SETTLEMENT || (found && state.FinalSettlementStarted) {
			k.EnsureSpotDelistCursor(ctx, pair.Id)
		}
	}
}
