package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestCreateSubaccount(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*k)
	owner := "dydx1x2hd82qerp7lc0kf5cs3yekftupkrl620te6u2"

	response, err := server.CreateSubaccount(sdk.WrapSDKContext(ctx), &types.MsgCreateSubaccount{
		Owner:       owner,
		AccountType: types.AccountType_ACCOUNT_TYPE_SPOT,
	})
	require.NoError(t, err)
	require.NotNil(t, response.SubaccountId)
	require.Equal(t, uint32(1), response.SubaccountId.Number)

	stored := k.GetSubaccount(ctx, *response.SubaccountId)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_SPOT, stored.AccountType)
	require.Empty(t, stored.AssetPositions)
	require.Empty(t, stored.PerpetualPositions)

	mappedID, exists := k.GetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT)
	require.True(t, exists)
	require.Equal(t, *response.SubaccountId, mappedID)
}

func TestCreateSubaccountRejectsInvalidAndDuplicateRequests(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	server := keeper.NewMsgServerImpl(*k)
	goCtx := sdk.WrapSDKContext(ctx)
	owner := "dydx1x2hd82qerp7lc0kf5cs3yekftupkrl620te6u2"

	_, err := server.CreateSubaccount(goCtx, &types.MsgCreateSubaccount{
		Owner:       owner,
		AccountType: types.AccountType_ACCOUNT_TYPE_PERPETUAL,
	})
	require.ErrorIs(t, err, types.ErrInvalidAccountType)
	require.Equal(t, uint32(1), k.GetNextBusinessSubaccountNumber(ctx, owner))

	_, err = server.CreateSubaccount(goCtx, &types.MsgCreateSubaccount{
		Owner:       owner,
		AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING,
	})
	require.NoError(t, err)

	_, err = server.CreateSubaccount(goCtx, &types.MsgCreateSubaccount{
		Owner:       owner,
		AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING,
	})
	require.ErrorIs(t, err, types.ErrAccountTypeAlreadyExists)
	require.Equal(t, uint32(2), k.GetNextBusinessSubaccountNumber(ctx, owner))
}
