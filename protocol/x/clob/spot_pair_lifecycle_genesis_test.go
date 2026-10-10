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

func TestSpotPairLifecycleGenesisRoundTrip(t *testing.T) {
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
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId:            assetId,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
	}

	genesis := types.DefaultGenesis()
	genesis.ClobPairs = []types.ClobPair{
		{
			Id:               4,
			StepBaseQuantums: 5,
			SubticksPerTick:  1,
			Status:           types.ClobPair_STATUS_ACTIVE,
			Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
				BaseAssetId: 1, QuoteAssetId: 0,
			}},
		},
		{
			Id:               9,
			StepBaseQuantums: 5,
			SubticksPerTick:  1,
			Status:           types.ClobPair_STATUS_FINAL_SETTLEMENT,
			Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
				BaseAssetId: 1, QuoteAssetId: 0,
			}},
		},
	}
	genesis.SpotPairLifecycles = []types.SpotPairLifecycle{
		{
			ClobPairId:             9,
			PauseSource:            types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_AUTO,
			FinalSettlementStarted: true,
		},
		{
			ClobPairId:  4,
			PauseSource: types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL,
		},
	}
	require.NoError(t, genesis.Validate())

	clob.InitGenesis(ks.Ctx, ks.ClobKeeper, *genesis)

	manual, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, 4)
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceManual, manual.PauseSource)
	require.False(t, manual.FinalSettlementStarted)
	finalized, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, 9)
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceAuto, finalized.PauseSource)
	require.True(t, finalized.FinalSettlementStarted)

	exported := clob.ExportGenesis(ks.Ctx, *ks.ClobKeeper)
	require.Equal(t, []types.SpotPairLifecycle{
		{
			ClobPairId:  4,
			PauseSource: types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL,
		},
		{
			ClobPairId:             9,
			PauseSource:            types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_AUTO,
			FinalSettlementStarted: true,
		},
	}, exported.SpotPairLifecycles)
}
