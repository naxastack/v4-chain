package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func activePolicy(assetId uint32) types.AssetPolicy {
	return types.AssetPolicy{
		AssetId:            assetId,
		Status:             types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		DepositsEnabled:    true,
		WithdrawalsEnabled: true,
		SpotTradingEnabled: true,
		PerpetualEnabled:   true,
	}
}

func TestValidateAssetPolicy(t *testing.T) {
	policy := activePolicy(1)
	require.NoError(t, types.ValidateAssetPolicy(policy))

	policy.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_UNSPECIFIED
	require.ErrorIs(t, types.ValidateAssetPolicy(policy), types.ErrInvalidAssetPolicyStatus)

	policy.Status = types.AssetPolicyStatus(99)
	require.ErrorIs(t, types.ValidateAssetPolicy(policy), types.ErrInvalidAssetPolicyStatus)

	policy = activePolicy(1)
	policy.WithdrawalsEnabled = false
	require.ErrorIs(t, types.ValidateAssetPolicy(policy), types.ErrAssetExitMustRemainEnabled)
}

func TestValidateAssetPolicyTransition(t *testing.T) {
	active := activePolicy(1)
	paused := active
	paused.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	retired := active
	retired.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED

	require.NoError(t, types.ValidateAssetPolicyTransition(active, paused))
	require.NoError(t, types.ValidateAssetPolicyTransition(paused, active))
	require.NoError(t, types.ValidateAssetPolicyTransition(active, retired))
	require.NoError(t, types.ValidateAssetPolicyTransition(retired, retired))
	require.ErrorIs(
		t,
		types.ValidateAssetPolicyTransition(retired, active),
		types.ErrInvalidAssetPolicyTransition,
	)

	otherAsset := activePolicy(2)
	require.ErrorIs(
		t,
		types.ValidateAssetPolicyTransition(active, otherAsset),
		types.ErrAssetPolicyAssetIdMismatch,
	)
}

func TestAssetPolicyPermissions(t *testing.T) {
	active := activePolicy(1)
	require.True(t, active.AllowsDeposit())
	require.True(t, active.AllowsWithdrawal())
	require.True(t, active.AllowsSpotTrading())
	require.True(t, active.AllowsPerpetual())

	paused := active
	paused.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.False(t, paused.AllowsDeposit())
	require.True(t, paused.AllowsWithdrawal())
	require.False(t, paused.AllowsSpotTrading())
	require.False(t, paused.AllowsPerpetual())

	retired := active
	retired.Status = types.AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED
	require.False(t, retired.AllowsDeposit())
	require.True(t, retired.AllowsWithdrawal())
	require.False(t, retired.AllowsSpotTrading())
	require.False(t, retired.AllowsPerpetual())
}
