package cli

import (
	"testing"

	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/stretchr/testify/require"
)

func TestSendingCommandsDefaultToUsdc(t *testing.T) {
	depositAsset, err := CmdDepositToSubaccount().Flags().GetUint32(flagAssetID)
	require.NoError(t, err)
	require.Equal(t, assettypes.AssetUsdc.Id, depositAsset)

	transferAsset, err := CmdCreateTransfer().Flags().GetUint32(flagAssetID)
	require.NoError(t, err)
	require.Equal(t, assettypes.AssetUsdc.Id, transferAsset)
}
