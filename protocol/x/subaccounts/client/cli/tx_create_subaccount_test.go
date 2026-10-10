package cli

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestParseBusinessAccountType(t *testing.T) {
	spot, err := parseBusinessAccountType("spot")
	require.NoError(t, err)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_SPOT, spot)

	funding, err := parseBusinessAccountType("FUNDING")
	require.NoError(t, err)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_FUNDING, funding)

	_, err = parseBusinessAccountType("perpetual")
	require.Error(t, err)
}
