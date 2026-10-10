package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateSpotClobPairIndexesBaseAndQuote(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	for _, asset := range []struct {
		id     uint32
		symbol string
		denom  string
	}{
		{1, "BASE", "ubase"},
		{2, "OTHER", "uother"},
	} {
		_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, asset.id, asset.symbol, asset.denom, 0, false, 0, 0)
		require.NoError(t, err)
	}
	for _, assetId := range []uint32{0, 1, 2} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId:            assetId,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
	}

	laterPair := spotPairForIndex(9, 2)
	earlierPair := spotPairForIndex(4, 1)
	laterPair.Status = types.ClobPair_STATUS_CANCEL_ONLY
	earlierPair.Status = types.ClobPair_STATUS_CANCEL_ONLY
	_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, laterPair)
	require.NoError(t, err)
	_, err = ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, earlierPair)
	require.NoError(t, err)

	require.Equal(t, []uint32{4, 9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 0))
	require.Equal(t, []uint32{4}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 1))
	require.Equal(t, []uint32{9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 2))
	require.Empty(t, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 3))

	_, err = ks.ClobKeeper.CreateSpotClobPairAndMemStructs(ks.Ctx, spotPairForIndex(7, 2))
	require.NoError(t, err)
	require.Equal(t, []uint32{7, 9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 2))
	require.Equal(t, []uint32{4, 7, 9}, ks.ClobKeeper.GetSpotClobPairIdsByAsset(ks.Ctx, 0))
}

func spotPairForIndex(id uint32, baseAssetId uint32) types.ClobPair {
	return types.ClobPair{
		Id: id,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: baseAssetId, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10,
		SubticksPerTick:  1,
		Status:           types.ClobPair_STATUS_ACTIVE,
	}
}
