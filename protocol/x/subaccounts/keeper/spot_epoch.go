package keeper

import (
	"encoding/binary"
	"math"
	"sort"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

func spotOrderEpochKey(id types.SubaccountId, assetID uint32) []byte {
	idKey := id.ToStateKey()
	key := make([]byte, 4+len(idKey)+4)
	binary.BigEndian.PutUint32(key[:4], uint32(len(idKey)))
	copy(key[4:], idKey)
	binary.BigEndian.PutUint32(key[4+len(idKey):], assetID)
	return key
}

// GetSpotOrderEpoch returns the current short-term order version for a SPOT asset.
func (k Keeper) GetSpotOrderEpoch(
	ctx sdk.Context,
	id types.SubaccountId,
	assetID uint32,
) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotOrderEpochKeyPrefix))
	value := store.Get(spotOrderEpochKey(id, assetID))
	if len(value) != 8 {
		return 0
	}
	return binary.BigEndian.Uint64(value)
}

// SetSpotOrderEpoch persists a non-zero short-term order version.
func (k Keeper) SetSpotOrderEpoch(
	ctx sdk.Context,
	id types.SubaccountId,
	assetID uint32,
	epoch uint64,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotOrderEpochKeyPrefix))
	key := spotOrderEpochKey(id, assetID)
	if epoch == 0 {
		store.Delete(key)
		return
	}
	value := make([]byte, 8)
	binary.BigEndian.PutUint64(value, epoch)
	store.Set(key, value)
}

// IncrementSpotOrderEpoch increments and returns the short-term order version.
func (k Keeper) IncrementSpotOrderEpoch(
	ctx sdk.Context,
	id types.SubaccountId,
	assetID uint32,
) (uint64, error) {
	epoch := k.GetSpotOrderEpoch(ctx, id, assetID)
	if epoch == math.MaxUint64 {
		return 0, types.ErrSpotOrderEpochOverflow
	}
	epoch++
	k.SetSpotOrderEpoch(ctx, id, assetID, epoch)
	return epoch, nil
}

// GetAllSpotOrderEpochs returns all persisted epochs in deterministic order.
func (k Keeper) GetAllSpotOrderEpochs(ctx sdk.Context) []types.SpotOrderEpoch {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotOrderEpochKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	result := make([]types.SpotOrderEpoch, 0)
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		if len(key) < 8 {
			continue
		}
		idLength := int(binary.BigEndian.Uint32(key[:4]))
		if idLength <= 0 || len(key) != 4+idLength+4 || len(iterator.Value()) != 8 {
			continue
		}
		var id types.SubaccountId
		if err := id.Unmarshal(key[4 : 4+idLength]); err != nil {
			continue
		}
		result = append(result, types.SpotOrderEpoch{
			SubaccountId: &id,
			AssetId:      binary.BigEndian.Uint32(key[4+idLength:]),
			Epoch:        binary.BigEndian.Uint64(iterator.Value()),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].SubaccountId
		right := result[j].SubaccountId
		if left.Owner != right.Owner {
			return left.Owner < right.Owner
		}
		if left.Number != right.Number {
			return left.Number < right.Number
		}
		return result[i].AssetId < result[j].AssetId
	})
	return result
}
