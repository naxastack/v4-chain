package clob_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// requirePerpetualSubaccountEqual compares a legacy perpetual subaccount fixture
// after accounting for explicit account types and canonical reservation zeros.
func requirePerpetualSubaccountEqual(
	t *testing.T,
	expected satypes.Subaccount,
	actual satypes.Subaccount,
) {
	t.Helper()

	if actual.AccountType == satypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED &&
		len(actual.AssetPositions) == 0 &&
		len(actual.PerpetualPositions) == 0 {
		// Deleted perpetual accounts retain the requested ID on read but have no
		// positions or persisted account metadata.
		expected.AccountType = satypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	} else if actual.AccountType == satypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		// USDC-only accounts are not promoted to PERPETUAL until a perpetual
		// position exists. They remain live and must still match positions.
		expected.AccountType = satypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	} else {
		require.Equal(t, satypes.AccountType_ACCOUNT_TYPE_PERPETUAL, actual.AccountType)
		expected.AccountType = satypes.AccountType_ACCOUNT_TYPE_PERPETUAL
	}

	expected.AssetPositions = append([]*satypes.AssetPosition(nil), expected.AssetPositions...)
	for i, position := range expected.AssetPositions {
		if position != nil && position.StatefulReservedQuantums.IsNil() {
			positionCopy := *position
			positionCopy.StatefulReservedQuantums = dtypes.ZeroInt()
			expected.AssetPositions[i] = &positionCopy
		}
	}

	require.Equal(t, expected, actual)
}
