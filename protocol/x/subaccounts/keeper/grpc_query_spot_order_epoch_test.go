package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestSpotOrderEpochAndSubaccountQuery(t *testing.T) {
	ctx, keeper, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, false)
	owner := constants.AliceAccAddress.String()
	id := types.SubaccountId{Owner: owner, Number: 2}
	keeper.SetSubaccount(ctx, types.Subaccount{
		Id:          &id,
		AccountType: types.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*types.AssetPosition{&types.AssetPosition{
			AssetId:                  1,
			Quantums:                 dtypes.NewInt(25),
			StatefulReservedQuantums: dtypes.NewInt(7),
		}},
	})
	keeper.SetSpotOrderEpoch(ctx, id, 1, 9)

	subaccount, err := keeper.Subaccount(ctx, &types.QueryGetSubaccountRequest{Owner: owner, Number: 2})
	require.NoError(t, err)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_SPOT, subaccount.Subaccount.AccountType)
	require.Equal(t, dtypes.NewInt(7), subaccount.Subaccount.AssetPositions[0].StatefulReservedQuantums)

	epoch, err := keeper.SpotOrderEpoch(ctx, &types.QuerySpotOrderEpochRequest{
		Owner:   owner,
		Number:  2,
		AssetId: 1,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(9), epoch.Epoch)

	missing, err := keeper.SpotOrderEpoch(ctx, &types.QuerySpotOrderEpochRequest{
		Owner:   owner,
		Number:  2,
		AssetId: 3,
	})
	require.NoError(t, err)
	require.Zero(t, missing.Epoch)
}
