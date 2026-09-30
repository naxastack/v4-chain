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

func TestCreateSpotClobPairValidatesAssetPolicies(t *testing.T) {
	newContext := func(t *testing.T) keepertest.ClobKeepersTestContext {
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
		return ks
	}

	pair := types.ClobPair{
		Id:                        100,
		Metadata:                  &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}},
		StepBaseQuantums:          10,
		QuantumConversionExponent: 0,
		SubticksPerTick:           1,
		Status:                    types.ClobPair_STATUS_ACTIVE,
	}

	t.Run("active policies", func(t *testing.T) {
		ks := newContext(t)
		created, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
		require.NoError(t, err)
		require.Equal(t, pair, created)
		stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
		require.True(t, found)
		require.Equal(t, pair, stored)
	})

	t.Run("non-USDC quote asset", func(t *testing.T) {
		ks := newContext(t)
		invalid := pair
		invalid.Metadata = &types.ClobPair_SpotClobMetadata{
			SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 2},
		}
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, invalid)
		require.ErrorContains(t, err, "must use asset 0 as quote asset")
	})
	t.Run("missing asset", func(t *testing.T) {
		ks := newContext(t)
		invalid := pair
		invalid.Metadata = &types.ClobPair_SpotClobMetadata{
			SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 2, QuoteAssetId: 0},
		}
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, invalid)
		require.ErrorContains(t, err, "Asset does not exist")
	})

	t.Run("trading disabled", func(t *testing.T) {
		ks := newContext(t)
		require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId:            1,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
		require.ErrorContains(t, err, "Asset spot trading is disabled")
	})
}
