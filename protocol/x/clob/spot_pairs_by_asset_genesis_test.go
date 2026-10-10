package clob_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSpotPairsByAssetRebuiltDuringGenesis(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	_, err = ks.AssetsKeeper.CreateAsset(ks.Ctx, 2, "OTHER", "uother", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1, 2} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId:            assetId,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
	}

	genesis := types.DefaultGenesis()
	genesis.ClobPairs = []types.ClobPair{
		spotGenesisPair(9, 2),
		spotGenesisPair(4, 1),
	}
	require.NoError(t, genesis.Validate())

	clob.InitGenesis(ks.Ctx, ks.ClobKeeper, *genesis)

	require.Equal(t, []uint32{4, 9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 0))
	require.Equal(t, []uint32{4}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 1))
	require.Equal(t, []uint32{9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 2))
}

func spotGenesisPair(id uint32, baseAssetId uint32) types.ClobPair {
	return types.ClobPair{
		Id:               id,
		StepBaseQuantums: 5,
		SubticksPerTick:  1,
		Status:           types.ClobPair_STATUS_ACTIVE,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: baseAssetId, QuoteAssetId: 0,
		}},
	}
}
