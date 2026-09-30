package keeper

import (
	"sort"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
)

// CreateAssetPolicy validates and stores the initial active policy for an asset.
func (k Keeper) CreateAssetPolicy(ctx sdk.Context, policy types.AssetPolicy) error {
	if _, exists := k.GetAsset(ctx, policy.AssetId); !exists {
		return errorsmod.Wrapf(types.ErrAssetDoesNotExist, "asset id = %d", policy.AssetId)
	}
	if _, exists := k.GetAssetPolicy(ctx, policy.AssetId); exists {
		return errorsmod.Wrapf(types.ErrAssetPolicyAlreadyExists, "asset id = %d", policy.AssetId)
	}
	if policy.Status != types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE {
		return types.ErrInvalidAssetPolicyStatus
	}
	if err := types.ValidateAssetPolicy(policy); err != nil {
		return err
	}
	k.setAssetPolicy(ctx, policy)
	return nil
}

// UpdateAssetPolicy validates and stores a policy update.
func (k Keeper) UpdateAssetPolicy(ctx sdk.Context, policy types.AssetPolicy) error {
	previous, exists := k.GetAssetPolicy(ctx, policy.AssetId)
	if !exists {
		return errorsmod.Wrapf(types.ErrAssetPolicyDoesNotExist, "asset id = %d", policy.AssetId)
	}
	if err := types.ValidateAssetPolicyTransition(previous, policy); err != nil {
		return err
	}
	k.setAssetPolicy(ctx, policy)
	return nil
}

// GetAssetPolicy returns the policy for an asset.
func (k Keeper) GetAssetPolicy(ctx sdk.Context, assetId uint32) (types.AssetPolicy, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyKeyPrefix))
	value := store.Get(lib.Uint32ToKey(assetId))
	if value == nil {
		return types.AssetPolicy{}, false
	}

	var policy types.AssetPolicy
	k.cdc.MustUnmarshal(value, &policy)
	return policy, true
}

// GetAllAssetPolicies returns policies ordered by asset id.
func (k Keeper) GetAllAssetPolicies(ctx sdk.Context) []types.AssetPolicy {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	policies := make([]types.AssetPolicy, 0)
	for ; iterator.Valid(); iterator.Next() {
		var policy types.AssetPolicy
		k.cdc.MustUnmarshal(iterator.Value(), &policy)
		policies = append(policies, policy)
	}
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].AssetId < policies[j].AssetId
	})
	return policies
}

func (k Keeper) setAssetPolicy(ctx sdk.Context, policy types.AssetPolicy) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyKeyPrefix))
	store.Set(lib.Uint32ToKey(policy.AssetId), k.cdc.MustMarshal(&policy))
}
