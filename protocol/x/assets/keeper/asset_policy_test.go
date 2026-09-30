package keeper_test

import (
	"testing"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func testAssetPolicy(assetId uint32) types.AssetPolicy {
	return types.AssetPolicy{
		AssetId:            assetId,
		Status:             types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		DepositsEnabled:    true,
		WithdrawalsEnabled: true,
		SpotTradingEnabled: true,
		PerpetualEnabled:   assetId == types.AssetUsdc.Id,
	}
}

func TestCreateAssetPolicy(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	policy := testAssetPolicy(types.AssetUsdc.Id)

	require.ErrorIs(t, keeper.CreateAssetPolicy(ctx, policy), types.ErrAssetDoesNotExist)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))
	require.NoError(t, keeper.CreateAssetPolicy(ctx, policy))

	stored, exists := keeper.GetAssetPolicy(ctx, policy.AssetId)
	require.True(t, exists)
	require.Equal(t, policy, stored)
	require.ErrorIs(t, keeper.CreateAssetPolicy(ctx, policy), types.ErrAssetPolicyAlreadyExists)
}

func TestCreateAssetPolicyRequiresActiveStatus(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))

	policy := testAssetPolicy(types.AssetUsdc.Id)
	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.ErrorIs(t, keeper.CreateAssetPolicy(ctx, policy), types.ErrInvalidAssetPolicyStatus)
	_, exists := keeper.GetAssetPolicy(ctx, policy.AssetId)
	require.False(t, exists)
}

func TestUpdateAssetPolicy(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))

	policy := testAssetPolicy(types.AssetUsdc.Id)
	require.NoError(t, keeper.CreateAssetPolicy(ctx, policy))

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED
	require.NoError(t, keeper.UpdateAssetPolicy(ctx, policy))

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE
	require.ErrorIs(t, keeper.UpdateAssetPolicy(ctx, policy), types.ErrInvalidAssetPolicyTransition)

	stored, exists := keeper.GetAssetPolicy(ctx, policy.AssetId)
	require.True(t, exists)
	require.Equal(t, types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED, stored.Status)
}

func TestGetAllAssetPoliciesOrdersByAssetId(t *testing.T) {
	ctx, keeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, keeper))
	require.NoError(t, keeper.CreateAssetPolicy(ctx, testAssetPolicy(types.AssetUsdc.Id)))

	policies := keeper.GetAllAssetPolicies(ctx)
	require.Len(t, policies, 1)
	require.Equal(t, types.AssetUsdc.Id, policies[0].AssetId)
}
