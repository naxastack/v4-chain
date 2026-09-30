package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisStateValidateStatefulSpotReservations(t *testing.T) {
	newGenesis := func() *types.GenesisState {
		genesis := types.DefaultGenesis()
		genesis.ClobPairs = []types.ClobPair{{
			Id: 7,
			Metadata: &types.ClobPair_SpotClobMetadata{
				SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0},
			},
		}}
		genesis.StatefulSpotReservations = []types.StatefulSpotReservation{{
			OrderId: types.OrderId{
				SubaccountId: satypes.SubaccountId{Owner: "owner", Number: 1},
				ClientId:     1,
				OrderFlags:   types.OrderIdFlags_LongTerm,
				ClobPairId:   7,
			},
			OutgoingAssetId:       0,
			ReservedQuantums:      dtypes.NewInt(10),
			RemainingBaseQuantums: 5,
		}}
		return genesis
	}

	t.Run("valid", func(t *testing.T) {
		require.NoError(t, newGenesis().Validate())
	})

	t.Run("duplicate order id", func(t *testing.T) {
		genesis := newGenesis()
		genesis.StatefulSpotReservations = append(
			genesis.StatefulSpotReservations,
			genesis.StatefulSpotReservations[0],
		)
		require.ErrorContains(t, genesis.Validate(), "duplicated stateful spot reservation order id")
	})

	t.Run("non-positive reservation", func(t *testing.T) {
		genesis := newGenesis()
		genesis.StatefulSpotReservations[0].ReservedQuantums = dtypes.NewInt(0)
		require.ErrorContains(t, genesis.Validate(), "must reserve positive quantums")
	})

	t.Run("zero remaining base", func(t *testing.T) {
		genesis := newGenesis()
		genesis.StatefulSpotReservations[0].RemainingBaseQuantums = 0
		require.ErrorContains(t, genesis.Validate(), "positive remaining base quantums")
	})

	t.Run("non-spot pair", func(t *testing.T) {
		genesis := newGenesis()
		genesis.ClobPairs[0].Metadata = &types.ClobPair_PerpetualClobMetadata{
			PerpetualClobMetadata: &types.PerpetualClobMetadata{PerpetualId: 1},
		}
		require.ErrorContains(t, genesis.Validate(), "must reference a spot CLOB pair")
	})

	t.Run("asset outside pair", func(t *testing.T) {
		genesis := newGenesis()
		genesis.StatefulSpotReservations[0].OutgoingAssetId = 2
		require.ErrorContains(t, genesis.Validate(), "outgoing asset must belong to its CLOB pair")
	})

	t.Run("short-term order", func(t *testing.T) {
		genesis := newGenesis()
		genesis.StatefulSpotReservations[0].OrderId.OrderFlags = types.OrderIdFlags_ShortTerm
		require.ErrorContains(t, genesis.Validate(), "must reference a long-term order")
	})
}
