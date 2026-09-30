package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisStateValidateAssetPolicies(t *testing.T) {
	tests := map[string]struct {
		policies    []types.AssetPolicy
		expectedErr error
	}{
		"valid active policy": {
			policies: []types.AssetPolicy{types.AssetPolicyUsdc},
		},
		"missing policy": {
			expectedErr: types.ErrAssetPolicyDoesNotExist,
		},
		"policy for unknown asset": {
			policies: []types.AssetPolicy{
				types.AssetPolicyUsdc,
				{
					AssetId:            1,
					Status:             types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
					WithdrawalsEnabled: true,
				},
			},
			expectedErr: types.ErrAssetDoesNotExist,
		},
		"duplicate policy": {
			policies: []types.AssetPolicy{
				types.AssetPolicyUsdc,
				types.AssetPolicyUsdc,
			},
			expectedErr: types.ErrAssetPolicyAlreadyExists,
		},
		"initial policy is paused": {
			policies: []types.AssetPolicy{
				{
					AssetId:            types.AssetUsdc.Id,
					Status:             types.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
					WithdrawalsEnabled: true,
				},
			},
			expectedErr: types.ErrInvalidAssetPolicyStatus,
		},
		"withdrawals disabled": {
			policies: []types.AssetPolicy{
				{
					AssetId: types.AssetUsdc.Id,
					Status:  types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
				},
			},
			expectedErr: types.ErrAssetExitMustRemainEnabled,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			genesis := types.GenesisState{
				Assets:        []types.Asset{types.AssetUsdc},
				AssetPolicies: tc.policies,
			}
			err := genesis.Validate()
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.expectedErr)
			}
		})
	}
}
