package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// indexSpotPairByAssets records a Spot pair under its Base and Quote assets.
// Perpetual pairs do not use this index. The index is rebuilt when pairs are
// created during genesis and is not stored as its own genesis field.
func (k Keeper) indexSpotPairByAssets(ctx sdk.Context, clobPair types.ClobPair) {
	metadata := clobPair.GetSpotClobMetadata()
	if metadata == nil {
		panic("spot CLOB metadata is required to index a spot pair")
	}
	k.setSpotPairForAsset(ctx, metadata.BaseAssetId, clobPair.Id)
	k.setSpotPairForAsset(ctx, metadata.QuoteAssetId, clobPair.Id)
}

func (k Keeper) setSpotPairForAsset(ctx sdk.Context, assetId uint32, clobPairId uint32) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), spotPairsByAssetPrefix(assetId))
	store.Set(lib.Uint32ToKey(clobPairId), []byte{1})
}

// GetSpotClobPairIdsByAsset returns Spot CLOB pair ids that reference assetId,
// ordered by pair id.
func (k Keeper) GetSpotClobPairIdsByAsset(ctx sdk.Context, assetId uint32) []uint32 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), spotPairsByAssetPrefix(assetId))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	pairIds := make([]uint32, 0)
	for ; iterator.Valid(); iterator.Next() {
		if len(iterator.Key()) != 4 {
			panic("invalid spot pairs-by-asset key")
		}
		pairIds = append(pairIds, lib.BytesToUint32(iterator.Key()))
	}
	return pairIds
}

func spotPairsByAssetPrefix(assetId uint32) []byte {
	return append([]byte(types.SpotPairsByAssetKeyPrefix), lib.Uint32ToKey(assetId)...)
}
