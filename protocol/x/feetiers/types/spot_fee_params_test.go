package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
	"github.com/stretchr/testify/require"
)

func TestSpotFeeParamsValidate(t *testing.T) {
	require.NoError(t, types.DefaultSpotFeeParams().Validate())
	require.Error(t, (types.SpotFeeParams{}).Validate())
	require.NoError(t, (types.SpotFeeParams{TradingFeePpm: types.MaxSpotTradingFeePpm}).Validate())
	require.Error(t, (types.SpotFeeParams{TradingFeePpm: types.MaxSpotTradingFeePpm + 1}).Validate())
}
