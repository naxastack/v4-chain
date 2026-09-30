package keeper_test

import (
	"testing"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func TestAssetPolicyValidationEntrypoints(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, testAssetPolicy(types.AssetUsdc.Id)))

	require.NoError(t, assetsKeeper.ValidateAssetDenom(
		ctx,
		types.AssetUsdc.Id,
		types.AssetUsdc.Denom,
	))
	require.NoError(t, assetsKeeper.ValidateAssetForDeposit(ctx, types.AssetUsdc.Id))
	require.NoError(t, assetsKeeper.ValidateAssetForWithdrawal(ctx, types.AssetUsdc.Id))
	require.NoError(t, assetsKeeper.ValidateAssetForSpotTrading(ctx, types.AssetUsdc.Id))
	require.NoError(t, assetsKeeper.ValidateAssetForPerpetual(ctx, types.AssetUsdc.Id))
}

func TestAssetPolicyValidationRejectsMissingAndMismatchedAssets(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)

	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForDeposit(ctx, 99),
		types.ErrAssetDoesNotExist,
	)

	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForDeposit(ctx, types.AssetUsdc.Id),
		types.ErrAssetPolicyDoesNotExist,
	)
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetDenom(ctx, types.AssetUsdc.Id, "uatom"),
		types.ErrAssetDenomMismatch,
	)
}

func TestAssetPolicyValidationHonorsLifecycleAndPermissions(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))

	policy := testAssetPolicy(types.AssetUsdc.Id)
	policy.DepositsEnabled = false
	policy.SpotTradingEnabled = false
	policy.PerpetualEnabled = false
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, policy))

	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForDeposit(ctx, policy.AssetId),
		types.ErrAssetDepositsDisabled,
	)
	require.NoError(t, assetsKeeper.ValidateAssetForWithdrawal(ctx, policy.AssetId))
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForSpotTrading(ctx, policy.AssetId),
		types.ErrAssetSpotTradingDisabled,
	)
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForPerpetual(ctx, policy.AssetId),
		types.ErrAssetPerpetualDisabled,
	)

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	policy.DepositsEnabled = true
	policy.SpotTradingEnabled = true
	policy.PerpetualEnabled = true
	require.NoError(t, assetsKeeper.UpdateAssetPolicy(ctx, policy))

	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForDeposit(ctx, policy.AssetId),
		types.ErrAssetDepositsDisabled,
	)
	require.NoError(t, assetsKeeper.ValidateAssetForWithdrawal(ctx, policy.AssetId))
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForSpotTrading(ctx, policy.AssetId),
		types.ErrAssetSpotTradingDisabled,
	)
	require.ErrorIs(
		t,
		assetsKeeper.ValidateAssetForPerpetual(ctx, policy.AssetId),
		types.ErrAssetPerpetualDisabled,
	)
}
