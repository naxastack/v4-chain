package keeper_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatefulSpotReservationStorageAndQueries(t *testing.T) {
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		&mocks.IndexerEventManager{},
	)
	owner := sampletest.AccAddress()
	firstId := types.OrderId{
		SubaccountId: satypes.SubaccountId{Owner: owner, Number: 1},
		ClientId:     1,
		OrderFlags:   types.OrderIdFlags_LongTerm,
		ClobPairId:   100,
	}
	secondId := firstId
	secondId.ClientId = 2
	for _, orderId := range []types.OrderId{firstId, secondId} {
		ks.ClobKeeper.SetStatefulSpotReservation(ks.Ctx, types.StatefulSpotReservation{
			OrderId:               orderId,
			OutgoingAssetId:       0,
			ReservedQuantums:      dtypes.NewInt(25),
			RemainingBaseQuantums: 10,
		})
	}

	response, err := ks.ClobKeeper.StatefulSpotReservation(
		ks.Ctx,
		&types.QueryStatefulSpotReservationRequest{OrderId: firstId},
	)
	require.NoError(t, err)
	require.Equal(t, firstId, response.Reservation.OrderId)
	require.Equal(t, int64(25), response.Reservation.ReservedQuantums.BigInt().Int64())

	page, err := ks.ClobKeeper.SpotStatefulOrdersByClobPair(
		ks.Ctx,
		&types.QuerySpotStatefulOrdersByClobPairRequest{
			ClobPairId: 100,
			Pagination: &query.PageRequest{Limit: 1, CountTotal: true},
		},
	)
	require.NoError(t, err)
	require.Len(t, page.OrderIds, 1)
	require.Equal(t, uint64(2), page.Pagination.Total)

	_, err = ks.ClobKeeper.StatefulSpotReservation(
		ks.Ctx,
		&types.QueryStatefulSpotReservationRequest{OrderId: types.OrderId{}},
	)
	require.Equal(t, codes.NotFound, status.Code(err))
}
