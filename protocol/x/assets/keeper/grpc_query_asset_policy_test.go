package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAssetPolicyQuery(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	policy := testAssetPolicy(types.AssetUsdc.Id)
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, policy))

	response, err := assetsKeeper.AssetPolicy(
		sdk.WrapSDKContext(ctx),
		&types.QueryAssetPolicyRequest{AssetId: policy.AssetId},
	)
	require.NoError(t, err)
	require.Equal(t, policy, response.Policy)
}

func TestAssetPolicyQueryNotFound(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	_, err := assetsKeeper.AssetPolicy(
		sdk.WrapSDKContext(ctx),
		&types.QueryAssetPolicyRequest{AssetId: 99},
	)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestAllAssetPoliciesQuery(t *testing.T) {
	ctx, assetsKeeper, _, _, _, _ := keepertest.AssetsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	policy := testAssetPolicy(types.AssetUsdc.Id)
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, policy))

	response, err := assetsKeeper.AllAssetPolicies(
		sdk.WrapSDKContext(ctx),
		&types.QueryAllAssetPoliciesRequest{},
	)
	require.NoError(t, err)
	require.Equal(t, []types.AssetPolicy{policy}, response.Policies)
}
