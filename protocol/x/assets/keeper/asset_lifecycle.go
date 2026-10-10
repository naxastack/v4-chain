package keeper

import (
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
)

func (k Keeper) GetAssetPolicyState(
	ctx sdk.Context,
	assetId uint32,
) (types.AssetPolicyState, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyStateKeyPrefix))
	value := store.Get(lib.Uint32ToKey(assetId))
	if value == nil {
		return types.AssetPolicyState{}, false
	}
	var state types.AssetPolicyState
	k.cdc.MustUnmarshal(value, &state)
	return state, true
}

func (k Keeper) GetAllAssetPolicyStates(ctx sdk.Context) []types.AssetPolicyState {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyStateKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	states := make([]types.AssetPolicyState, 0)
	for ; iterator.Valid(); iterator.Next() {
		var state types.AssetPolicyState
		k.cdc.MustUnmarshal(iterator.Value(), &state)
		states = append(states, state)
	}
	return states
}

func (k Keeper) setAssetPolicyState(ctx sdk.Context, state types.AssetPolicyState) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyStateKeyPrefix))
	store.Set(lib.Uint32ToKey(state.AssetId), k.cdc.MustMarshal(&state))
}

func (k Keeper) GetAssetLifecycleTask(
	ctx sdk.Context,
	assetId uint32,
) (types.AssetLifecycleTask, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetLifecycleTaskKeyPrefix))
	value := store.Get(lib.Uint32ToKey(assetId))
	if value == nil {
		return types.AssetLifecycleTask{}, false
	}
	var task types.AssetLifecycleTask
	k.cdc.MustUnmarshal(value, &task)
	return task, true
}

func (k Keeper) GetAllAssetLifecycleTasks(ctx sdk.Context) []types.AssetLifecycleTask {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetLifecycleTaskKeyPrefix))
	iterator := storetypes.KVStorePrefixIterator(store, nil)
	defer iterator.Close()

	tasks := make([]types.AssetLifecycleTask, 0)
	for ; iterator.Valid(); iterator.Next() {
		var task types.AssetLifecycleTask
		k.cdc.MustUnmarshal(iterator.Value(), &task)
		tasks = append(tasks, task)
	}
	return tasks
}

// GetActionableAssetLifecycleTasks returns incomplete tasks whose target
// version still matches the asset lifecycle version, ordered by asset id.
func (k Keeper) GetActionableAssetLifecycleTasks(ctx sdk.Context) []types.AssetLifecycleTask {
	tasks := make([]types.AssetLifecycleTask, 0)
	for _, task := range k.GetAllAssetLifecycleTasks(ctx) {
		if task.Completed {
			continue
		}
		state, found := k.GetAssetPolicyState(ctx, task.AssetId)
		if !found || state.LifecycleVersion != task.TargetLifecycleVersion {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (k Keeper) setAssetLifecycleTask(ctx sdk.Context, task types.AssetLifecycleTask) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetLifecycleTaskKeyPrefix))
	store.Set(lib.Uint32ToKey(task.AssetId), k.cdc.MustMarshal(&task))
}

// SetAssetLifecycleTaskProgress updates scheduler progress for the task whose
// lifecycle version is expectedVersion. A mismatched version does not write.
func (k Keeper) SetAssetLifecycleTaskProgress(
	ctx sdk.Context,
	assetId uint32,
	expectedLifecycleVersion uint64,
	nextClobPairId uint32,
	completed bool,
) error {
	task, found := k.GetAssetLifecycleTask(ctx, assetId)
	if !found {
		return types.ErrAssetLifecycleTaskDoesNotExist
	}
	state, found := k.GetAssetPolicyState(ctx, assetId)
	if !found ||
		state.LifecycleVersion != expectedLifecycleVersion ||
		task.TargetLifecycleVersion != expectedLifecycleVersion {
		return types.ErrStaleAssetLifecycleVersion
	}
	task.NextClobPairId = nextClobPairId
	task.Completed = completed
	k.setAssetLifecycleTask(ctx, task)
	return nil
}

func (k Keeper) GetAssetLifecycleSchedulerCursor(
	ctx sdk.Context,
) types.AssetLifecycleSchedulerCursor {
	store := ctx.KVStore(k.storeKey)
	value := store.Get([]byte(types.AssetLifecycleSchedulerCursorKey))
	if value == nil {
		return types.AssetLifecycleSchedulerCursor{}
	}
	var cursor types.AssetLifecycleSchedulerCursor
	k.cdc.MustUnmarshal(value, &cursor)
	return cursor
}

func (k Keeper) SetAssetLifecycleSchedulerCursor(
	ctx sdk.Context,
	cursor types.AssetLifecycleSchedulerCursor,
) {
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(types.AssetLifecycleSchedulerCursorKey), k.cdc.MustMarshal(&cursor))
}

// ImportGenesisAssetPolicy stores a genesis policy without changing versions.
// Active policies use the normal create path and start at version 1.
func (k Keeper) ImportGenesisAssetPolicy(ctx sdk.Context, policy types.AssetPolicy) error {
	if policy.Status == types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE {
		return k.CreateAssetPolicy(ctx, policy)
	}
	if _, exists := k.GetAsset(ctx, policy.AssetId); !exists {
		return types.ErrAssetDoesNotExist
	}
	if _, exists := k.GetAssetPolicy(ctx, policy.AssetId); exists {
		return types.ErrAssetPolicyAlreadyExists
	}
	if err := types.ValidateAssetPolicy(policy); err != nil {
		return err
	}
	k.setAssetPolicy(ctx, policy)
	return nil
}

func (k Keeper) ImportAssetPolicyState(ctx sdk.Context, state types.AssetPolicyState) error {
	if _, exists := k.GetAssetPolicy(ctx, state.AssetId); !exists {
		return types.ErrAssetPolicyDoesNotExist
	}
	if state.PolicyVersion == 0 || state.LifecycleVersion == 0 {
		return types.ErrInvalidAssetLifecycleTask
	}
	k.setAssetPolicyState(ctx, state)
	return nil
}

func (k Keeper) ImportAssetLifecycleTask(ctx sdk.Context, task types.AssetLifecycleTask) error {
	state, found := k.GetAssetPolicyState(ctx, task.AssetId)
	if !found || task.TargetLifecycleVersion != state.LifecycleVersion {
		return types.ErrInvalidAssetLifecycleTask
	}
	switch task.TargetStatus {
	case types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
		types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED:
	default:
		return types.ErrInvalidAssetLifecycleTask
	}
	k.setAssetLifecycleTask(ctx, task)
	return nil
}
