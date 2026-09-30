package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func createAssetMessage(assetId uint32) *types.MsgCreateAsset {
	return &types.MsgCreateAsset{
		Authority: constants.GovAuthority,
		Asset: types.Asset{
			Id:               assetId,
			Symbol:           "ATOM",
			Denom:            "uatom",
			DenomExponent:    -6,
			AtomicResolution: -6,
		},
		Policy: types.AssetPolicy{
			AssetId:            assetId,
			Status:             types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			DepositsEnabled:    true,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		},
	}
}

func TestMsgServerCreateAsset(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*assetsKeeper)
	goCtx := sdk.WrapSDKContext(ctx)

	msg := createAssetMessage(1)
	_, err := server.CreateAsset(goCtx, msg)
	require.NoError(t, err)

	asset, exists := assetsKeeper.GetAsset(ctx, msg.Asset.Id)
	require.True(t, exists)
	require.Equal(t, msg.Asset, asset)
	policy, exists := assetsKeeper.GetAssetPolicy(ctx, msg.Policy.AssetId)
	require.True(t, exists)
	require.Equal(t, msg.Policy, policy)
}

func TestMsgServerCreateAssetRejectsUnauthorized(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*assetsKeeper)
	goCtx := sdk.WrapSDKContext(ctx)

	msg := createAssetMessage(1)
	msg.Authority = "unauthorized"
	_, err := server.CreateAsset(goCtx, msg)
	require.Error(t, err)
	_, assetExists := assetsKeeper.GetAsset(ctx, msg.Asset.Id)
	_, policyExists := assetsKeeper.GetAssetPolicy(ctx, msg.Asset.Id)
	require.False(t, assetExists)
	require.False(t, policyExists)
}

func TestMsgServerCreateAssetRollsBackWhenPolicyFails(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*assetsKeeper)
	goCtx := sdk.WrapSDKContext(ctx)

	msg := createAssetMessage(1)
	msg.Policy.WithdrawalsEnabled = false
	_, err := server.CreateAsset(goCtx, msg)
	require.ErrorIs(t, err, types.ErrAssetExitMustRemainEnabled)

	_, assetExists := assetsKeeper.GetAsset(ctx, msg.Asset.Id)
	_, policyExists := assetsKeeper.GetAssetPolicy(ctx, msg.Asset.Id)
	require.False(t, assetExists)
	require.False(t, policyExists)
}

func TestMsgServerUpdateAssetPolicy(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*assetsKeeper)
	goCtx := sdk.WrapSDKContext(ctx)

	createMsg := createAssetMessage(1)
	_, err := server.CreateAsset(goCtx, createMsg)
	require.NoError(t, err)

	policy := createMsg.Policy
	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	_, err = server.UpdateAssetPolicy(goCtx, &types.MsgUpdateAssetPolicy{
		Authority: constants.GovAuthority,
		Policy:    policy,
	})
	require.NoError(t, err)

	stored, exists := assetsKeeper.GetAssetPolicy(ctx, policy.AssetId)
	require.True(t, exists)
	require.Equal(t, policy, stored)
}

func TestMsgServerUpdateAssetPolicyRejectsUnauthorized(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*assetsKeeper)
	goCtx := sdk.WrapSDKContext(ctx)

	_, err := server.UpdateAssetPolicy(goCtx, &types.MsgUpdateAssetPolicy{
		Authority: "unauthorized",
		Policy:    testAssetPolicy(1),
	})
	require.Error(t, err)
}
