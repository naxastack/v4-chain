package keeper_test

import (
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/indexer/indexer_manager"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestGetSpotCandidateState(t *testing.T) {
	ks := keeper.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer_manager.NewIndexerEventManagerNoop(),
	)
	pair := types.ClobPair{
		Id: 7,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 5, SubticksPerTick: 1,
	}
	ks.ClobKeeper.SetClobPair(ks.Ctx, pair)
	id := satypes.SubaccountId{Owner: "dydx1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp8a3s", Number: 1}
	ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
		Id: &id, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{
			{AssetId: 0, Quantums: dtypes.NewInt(100), StatefulReservedQuantums: dtypes.NewInt(22)},
			{AssetId: 1, Quantums: dtypes.NewInt(10), StatefulReservedQuantums: dtypes.NewInt(3)},
		},
	})
	ks.SubaccountsKeeper.SetSpotOrderEpoch(ks.Ctx, id, 0, 4)
	ks.SubaccountsKeeper.SetSpotOrderEpoch(ks.Ctx, id, 1, 5)

	order := types.Order{OrderId: types.OrderId{SubaccountId: id, ClobPairId: 7}, Side: types.Order_SIDE_BUY}
	state, err := ks.ClobKeeper.GetSpotCandidateState(ks.Ctx, order)
	require.NoError(t, err)
	require.Equal(t, uint32(0), state.OutgoingAssetId)
	require.Equal(t, big.NewInt(100), state.TotalQuantums)
	require.Equal(t, big.NewInt(22), state.StatefulReservedQuantums)
	require.Equal(t, uint64(4), state.CurrentOrderEpoch)

	order.Side = types.Order_SIDE_SELL
	state, err = ks.ClobKeeper.GetSpotCandidateState(ks.Ctx, order)
	require.NoError(t, err)
	require.Equal(t, uint32(1), state.OutgoingAssetId)
	require.Equal(t, big.NewInt(10), state.TotalQuantums)
	require.Equal(t, big.NewInt(3), state.StatefulReservedQuantums)
	require.Equal(t, uint64(5), state.CurrentOrderEpoch)
}

func TestGetSpotCandidateStateMissingPositionReturnsZeroBalance(t *testing.T) {
	ks := keeper.NewClobKeepersTestContext(t, memclob.NewMemClobPriceTimePriority(false), &mocks.BankKeeper{}, indexer_manager.NewIndexerEventManagerNoop())
	ks.ClobKeeper.SetClobPair(ks.Ctx, types.ClobPair{Id: 7, Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}}})
	id := satypes.SubaccountId{Owner: "dydx1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqp8a3s", Number: 1}
	state, err := ks.ClobKeeper.GetSpotCandidateState(ks.Ctx, types.Order{OrderId: types.OrderId{SubaccountId: id, ClobPairId: 7}, Side: types.Order_SIDE_BUY})
	require.NoError(t, err)
	require.Zero(t, state.TotalQuantums.Sign())
	require.Zero(t, state.StatefulReservedQuantums.Sign())
}
