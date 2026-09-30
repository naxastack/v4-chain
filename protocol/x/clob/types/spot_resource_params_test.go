package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func TestSpotResourceParamsValidate(t *testing.T) {
	require.NoError(t, types.DefaultSpotResourceParams().Validate())

	for _, params := range []types.SpotResourceParams{
		{MaxSpotStatefulOrdersPerSubaccount: 0, MaxSpotDelistOrdersPerBlock: 1},
		{MaxSpotStatefulOrdersPerSubaccount: 1, MaxSpotDelistOrdersPerBlock: 0},
		{MaxSpotStatefulOrdersPerSubaccount: types.MaxSpotResourceLimit + 1, MaxSpotDelistOrdersPerBlock: 1},
		{MaxSpotStatefulOrdersPerSubaccount: 1, MaxSpotDelistOrdersPerBlock: types.MaxSpotResourceLimit + 1},
	} {
		require.Error(t, params.Validate())
	}

	require.NoError(t, (types.SpotResourceParams{
		MaxSpotStatefulOrdersPerSubaccount: types.MaxSpotResourceLimit,
		MaxSpotDelistOrdersPerBlock:        types.MaxSpotResourceLimit,
	}).Validate())
}
