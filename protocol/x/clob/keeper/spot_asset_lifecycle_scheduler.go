package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// AdvanceSpotAssetLifecycleTasks applies actionable asset tasks to Spot pairs.
// Each task advances at most one pair before the scheduler rotates to the next
// asset. The same pair is advanced at most once in a block.
func (k Keeper) AdvanceSpotAssetLifecycleTasks(ctx sdk.Context) {
	tasks := k.assetsKeeper.GetActionableAssetLifecycleTasks(ctx)
	if len(tasks) == 0 {
		return
	}
	limit := k.GetSpotResourceParams(ctx).MaxAssetLifecyclePairsPerBlock
	if limit == 0 {
		return
	}

	cursor := k.assetsKeeper.GetAssetLifecycleSchedulerCursor(ctx).NextAssetId
	start := 0
	for i, task := range tasks {
		if task.AssetId >= cursor {
			start = i
			break
		}
	}

	seen := make(map[uint32]struct{})
	advanced := uint32(0)
	idle := 0
	idx := start
	for advanced < limit && idle < len(tasks) {
		if k.advanceOneSpotAssetLifecyclePair(ctx, tasks[idx].AssetId, seen) {
			advanced++
			idle = 0
			next := tasks[(idx+1)%len(tasks)].AssetId
			k.assetsKeeper.SetAssetLifecycleSchedulerCursor(ctx, assettypes.AssetLifecycleSchedulerCursor{
				NextAssetId: next,
			})
		} else {
			idle++
		}
		idx = (idx + 1) % len(tasks)
	}
}

func (k Keeper) advanceOneSpotAssetLifecyclePair(
	ctx sdk.Context,
	assetId uint32,
	seen map[uint32]struct{},
) bool {
	task, found := k.assetsKeeper.GetAssetLifecycleTask(ctx, assetId)
	if !found || task.Completed {
		return false
	}
	state, found := k.assetsKeeper.GetAssetPolicyState(ctx, assetId)
	if !found || state.LifecycleVersion != task.TargetLifecycleVersion {
		return false
	}
	target, ok := clobStatusForAssetLifecycle(task.TargetStatus)
	if !ok {
		return false
	}

	pairIds := k.GetSpotClobPairIdsByAsset(ctx, assetId)
	for _, pairId := range pairIds {
		if pairId < task.NextClobPairId {
			continue
		}
		if k.spotPairSatisfiesAssetLifecycleTarget(ctx, pairId, target) {
			continue
		}
		if _, alreadyAdvanced := seen[pairId]; alreadyAdvanced {
			_ = k.assetsKeeper.SetAssetLifecycleTaskProgress(
				ctx,
				assetId,
				state.LifecycleVersion,
				pairId,
				false,
			)
			return false
		}
		if err := k.AdvanceSpotAssetLifecycle(ctx, types.ClobPairId(pairId), target); err != nil {
			_ = k.assetsKeeper.SetAssetLifecycleTaskProgress(
				ctx,
				assetId,
				state.LifecycleVersion,
				pairId,
				false,
			)
			return false
		}
		seen[pairId] = struct{}{}
		nextPairId, completed := k.nextSpotAssetLifecycleCursor(ctx, pairIds, pairId, target)
		if err := k.assetsKeeper.SetAssetLifecycleTaskProgress(
			ctx,
			assetId,
			state.LifecycleVersion,
			nextPairId,
			completed,
		); err != nil {
			return false
		}
		return true
	}

	_ = k.assetsKeeper.SetAssetLifecycleTaskProgress(ctx, assetId, state.LifecycleVersion, 0, true)
	return false
}

func (k Keeper) nextSpotAssetLifecycleCursor(
	ctx sdk.Context,
	pairIds []uint32,
	after uint32,
	target types.ClobPair_Status,
) (uint32, bool) {
	next := uint32(0)
	completed := true
	for _, pairId := range pairIds {
		if k.spotPairSatisfiesAssetLifecycleTarget(ctx, pairId, target) {
			continue
		}
		completed = false
		if pairId > after && next == 0 {
			next = pairId
		}
	}
	return next, completed
}

func (k Keeper) spotPairSatisfiesAssetLifecycleTarget(
	ctx sdk.Context,
	clobPairId uint32,
	target types.ClobPair_Status,
) bool {
	pair, found := k.GetClobPair(ctx, types.ClobPairId(clobPairId))
	if !found || pair.GetSpotClobMetadata() == nil {
		return true
	}
	state, _ := k.GetSpotPairLifecycleState(ctx, pair.GetClobPairId())
	switch target {
	case types.ClobPair_STATUS_ACTIVE:
		if pair.Status == types.ClobPair_STATUS_ACTIVE ||
			pair.Status == types.ClobPair_STATUS_FINAL_SETTLEMENT {
			return true
		}
		return pair.Status == types.ClobPair_STATUS_CANCEL_ONLY &&
			state.PauseSource == types.SpotPauseSourceManual
	case types.ClobPair_STATUS_CANCEL_ONLY:
		return pair.Status == types.ClobPair_STATUS_CANCEL_ONLY ||
			pair.Status == types.ClobPair_STATUS_FINAL_SETTLEMENT
	case types.ClobPair_STATUS_FINAL_SETTLEMENT:
		return pair.Status == types.ClobPair_STATUS_FINAL_SETTLEMENT
	default:
		return true
	}
}

func clobStatusForAssetLifecycle(status assettypes.AssetPolicyStatus) (types.ClobPair_Status, bool) {
	switch status {
	case assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE:
		return types.ClobPair_STATUS_ACTIVE, true
	case assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED:
		return types.ClobPair_STATUS_CANCEL_ONLY, true
	case assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED:
		return types.ClobPair_STATUS_FINAL_SETTLEMENT, true
	default:
		return 0, false
	}
}
