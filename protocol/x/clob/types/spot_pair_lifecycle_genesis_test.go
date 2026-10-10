package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisStateValidateSpotPairLifecycles(t *testing.T) {
	newGenesis := func() *types.GenesisState {
		genesis := types.DefaultGenesis()
		genesis.ClobPairs = []types.ClobPair{
			{
				Id: 7,
				Metadata: &types.ClobPair_SpotClobMetadata{
					SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0},
				},
			},
			{
				Id: 8,
				Metadata: &types.ClobPair_PerpetualClobMetadata{
					PerpetualClobMetadata: &types.PerpetualClobMetadata{PerpetualId: 1},
				},
			},
		}
		genesis.SpotPairLifecycles = []types.SpotPairLifecycle{{
			ClobPairId:             7,
			PauseSource:            types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL,
			FinalSettlementStarted: false,
		}}
		return genesis
	}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, newGenesis().Validate())
	})

	t.Run("duplicate pair", func(t *testing.T) {
		genesis := newGenesis()
		genesis.SpotPairLifecycles = append(genesis.SpotPairLifecycles, genesis.SpotPairLifecycles[0])
		require.ErrorContains(t, genesis.Validate(), "duplicated spot pair lifecycle")
	})

	t.Run("unknown pair", func(t *testing.T) {
		genesis := newGenesis()
		genesis.SpotPairLifecycles[0].ClobPairId = 9
		require.ErrorContains(t, genesis.Validate(), "unknown clob pair")
	})

	t.Run("non-spot pair", func(t *testing.T) {
		genesis := newGenesis()
		genesis.SpotPairLifecycles[0].ClobPairId = 8
		require.ErrorContains(t, genesis.Validate(), "must reference a spot CLOB pair")
	})

	t.Run("invalid pause source", func(t *testing.T) {
		genesis := newGenesis()
		genesis.SpotPairLifecycles[0].PauseSource = 9
		require.ErrorContains(t, genesis.Validate(), "invalid spot pair pause source")
	})
}
