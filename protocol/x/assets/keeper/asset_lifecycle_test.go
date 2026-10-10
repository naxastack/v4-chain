package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateAssetPolicyVersionsAndTasks(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))
	policy := testAssetPolicy(types.AssetUsdc.Id)
	require.NoError(t, keeper.CreateAssetPolicy(ctx, policy))

	state, found := keeper.GetAssetPolicyState(ctx, policy.AssetId)
	require.True(t, found)
	require.Equal(t, uint64(1), state.PolicyVersion)
	require.Equal(t, uint64(1), state.LifecycleVersion)
	_, found = keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.False(t, found)

	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	state, found = keeper.GetAssetPolicyState(ctx, policy.AssetId)
	require.True(t, found)
	require.Equal(t, uint64(1), state.PolicyVersion)
	require.Equal(t, uint64(1), state.LifecycleVersion)

	policy.DepositsEnabled = false
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	state, _ = keeper.GetAssetPolicyState(ctx, policy.AssetId)
	require.Equal(t, uint64(2), state.PolicyVersion)
	require.Equal(t, uint64(1), state.LifecycleVersion)
	_, found = keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.False(t, found)

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	state, _ = keeper.GetAssetPolicyState(ctx, policy.AssetId)
	require.Equal(t, uint64(3), state.PolicyVersion)
	require.Equal(t, uint64(2), state.LifecycleVersion)
	task, found := keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.True(t, found)
	require.Equal(t, uint64(2), task.TargetLifecycleVersion)
	require.Equal(t, types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED, task.TargetStatus)
	require.False(t, task.Completed)

	policy.WithdrawalsEnabled = true
	policy.SpotTradingEnabled = false
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	preserved, found := keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.True(t, found)
	require.Equal(t, task, preserved)
	state, _ = keeper.GetAssetPolicyState(ctx, policy.AssetId)
	require.Equal(t, uint64(4), state.PolicyVersion)
	require.Equal(t, uint64(2), state.LifecycleVersion)

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	require.ErrorIs(t, keeper.SetAssetLifecycleTaskProgress(ctx, policy.AssetId, 2, 9, true), types.ErrStaleAssetLifecycleVersion)
	stale, _ := keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.Equal(t, uint64(3), stale.TargetLifecycleVersion)
	require.Equal(t, types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED, stale.TargetStatus)
	require.False(t, stale.Completed)

	require.NoError(t, keeper.SetAssetLifecycleTaskProgress(ctx, policy.AssetId, 3, 4, false))
	updated, _ := keeper.GetAssetLifecycleTask(ctx, policy.AssetId)
	require.Equal(t, uint32(4), updated.NextClobPairId)
	require.False(t, updated.Completed)
}

func TestAssetLifecycleGenesisRoundTrip(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))
	_, err := keeper.CreateAsset(ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, keeper.CreateAssetPolicy(ctx, testAssetPolicy(assetId)))
	}
	paused := testAssetPolicy(1)
	paused.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, paused))
	require.NoError(t, keeper.SetAssetLifecycleTaskProgress(ctx, 1, 2, 7, false))
	keeper.SetAssetLifecycleSchedulerCursor(ctx, types.AssetLifecycleSchedulerCursor{NextAssetId: 1})

	exported := assets.ExportGenesis(ctx, *keeper)
	require.NoError(t, exported.Validate())
	ctx2, keeper2, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	assets.InitGenesis(ctx2, *keeper2, *exported)

	require.Equal(t, exported.AssetPolicyStates, keeper2.GetAllAssetPolicyStates(ctx2))
	require.Equal(t, exported.AssetLifecycleTasks, keeper2.GetAllAssetLifecycleTasks(ctx2))
	require.Equal(t, exported.AssetLifecycleSchedulerCursor, keeper2.GetAssetLifecycleSchedulerCursor(ctx2))
}

func TestAssetLifecycleQueries(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))
	policy := testAssetPolicy(types.AssetUsdc.Id)
	require.NoError(t, keeper.CreateAssetPolicy(ctx, policy))
	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))
	keeper.SetAssetLifecycleSchedulerCursor(ctx, types.AssetLifecycleSchedulerCursor{NextAssetId: 4})

	stateResponse, err := keeper.AssetPolicyState(
		sdk.WrapSDKContext(ctx),
		&types.QueryAssetPolicyStateRequest{AssetId: policy.AssetId},
	)
	require.NoError(t, err)
	require.Equal(t, uint64(2), stateResponse.State.LifecycleVersion)

	taskResponse, err := keeper.AssetLifecycleTask(
		sdk.WrapSDKContext(ctx),
		&types.QueryAssetLifecycleTaskRequest{AssetId: policy.AssetId},
	)
	require.NoError(t, err)
	require.Equal(t, types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED, taskResponse.Task.TargetStatus)

	cursorResponse, err := keeper.AssetLifecycleSchedulerCursor(
		sdk.WrapSDKContext(ctx),
		&types.QueryAssetLifecycleSchedulerCursorRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, uint32(4), cursorResponse.Cursor.NextAssetId)
}
