package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestAdvanceSpotAssetLifecycleTasksRotatesAcrossAssets(t *testing.T) {
	ks := newSpotLifecycleKeeper(t)
	for _, asset := range []struct {
		id     uint32
		symbol string
		denom  string
	}{
		{1, "ONE", "uone"},
		{2, "TWO", "utwo"},
		{3, "THREE", "uthree"},
	} {
		_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, asset.id, asset.symbol, asset.denom, 0, false, 0, 0)
		require.NoError(t, err)
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, activePolicy(asset.id)))
	}
	for _, pair := range []struct {
		id   uint32
		base uint32
	}{
		{10, 1}, {11, 1}, {12, 1}, {20, 2}, {30, 3},
	} {
		createActiveSpotClobPair(t, ks, spotPairForIndex(pair.id, pair.base))
	}
	for _, assetId := range []uint32{1, 2, 3} {
		policy := activePolicy(assetId)
		policy.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
		require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, policy))
	}
	params := types.DefaultSpotResourceParams()
	params.MaxAssetLifecyclePairsPerBlock = 3
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, params))

	ks.ClobKeeper.AdvanceSpotAssetLifecycleTasks(ks.Ctx)
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 10))
	require.Equal(t, types.ClobPair_STATUS_ACTIVE, spotStatus(t, ks, 11))
	require.Equal(t, types.ClobPair_STATUS_ACTIVE, spotStatus(t, ks, 12))
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 20))
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 30))
	require.Equal(t, uint32(11), lifecycleTask(t, ks, 1).NextClobPairId)
	require.True(t, lifecycleTask(t, ks, 2).Completed)
	require.True(t, lifecycleTask(t, ks, 3).Completed)

	ks.ClobKeeper.AdvanceSpotAssetLifecycleTasks(ks.Ctx)
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 11))
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 12))
	require.True(t, lifecycleTask(t, ks, 1).Completed)
	state, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, 10)
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceAuto, state.PauseSource)
}

func TestAdvanceSpotAssetLifecycleTasksDedupesSharedPair(t *testing.T) {
	ks := newSpotLifecycleKeeper(t)
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, activePolicy(1)))
	createActiveSpotClobPair(t, ks, spotPairForIndex(4, 1))
	for _, assetId := range []uint32{0, 1} {
		policy := activePolicy(assetId)
		policy.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
		require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, policy))
	}
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))

	ks.ClobKeeper.AdvanceSpotAssetLifecycleTasks(ks.Ctx)

	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 4))
	require.True(t, lifecycleTask(t, ks, 0).Completed)
	require.True(t, lifecycleTask(t, ks, 1).Completed)
	state, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, 4)
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceAuto, state.PauseSource)
	require.False(t, state.FinalSettlementStarted)
}

func TestAdvanceSpotAssetLifecycleTasksPreservesManualPause(t *testing.T) {
	ks := newSpotLifecycleKeeper(t)
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, activePolicy(1)))
	pair := spotPairForIndex(4, 1)
	pair.Status = types.ClobPair_STATUS_CANCEL_ONLY
	_, err = ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
	require.NoError(t, err)
	ks.ClobKeeper.SetSpotPairLifecycleState(ks.Ctx, pair.GetClobPairId(), types.SpotPairLifecycleState{
		PauseSource: types.SpotPauseSourceManual,
	})
	paused := activePolicy(1)
	paused.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, paused))
	require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, activePolicy(1)))
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))

	ks.ClobKeeper.AdvanceSpotAssetLifecycleTasks(ks.Ctx)

	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, spotStatus(t, ks, 4))
	state, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, 4)
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceManual, state.PauseSource)
	require.True(t, lifecycleTask(t, ks, 1).Completed)
}

func newSpotLifecycleKeeper(t *testing.T) keepertest.ClobKeepersTestContext {
	t.Helper()
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, activePolicy(0)))
	configureSpotActivationParams(t, ks)
	return ks
}

func activePolicy(assetId uint32) assettypes.AssetPolicy {
	return assettypes.AssetPolicy{
		AssetId:            assetId,
		Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		DepositsEnabled:    true,
		WithdrawalsEnabled: true,
		SpotTradingEnabled: true,
		PerpetualEnabled:   assetId == 0,
	}
}

func spotStatus(t *testing.T, ks keepertest.ClobKeepersTestContext, pairId uint32) types.ClobPair_Status {
	t.Helper()
	pair, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pairId))
	require.True(t, found)
	return pair.Status
}

func lifecycleTask(
	t *testing.T,
	ks keepertest.ClobKeepersTestContext,
	assetId uint32,
) assettypes.AssetLifecycleTask {
	t.Helper()
	task, found := ks.AssetsKeeper.GetAssetLifecycleTask(ks.Ctx, assetId)
	require.True(t, found)
	return task
}
