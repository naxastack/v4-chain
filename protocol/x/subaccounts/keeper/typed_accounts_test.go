package keeper_test

import (
	"math/big"
	"testing"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	testutil "github.com/dydxprotocol/v4-chain/protocol/testutil/util"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestAllocateBusinessSubaccount(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	owner := "dydx1x2hd82qerp7lc0kf5cs3yekftupkrl620te6u2"

	spotID, err := k.AllocateBusinessSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT)
	require.NoError(t, err)
	require.Equal(t, uint32(1), spotID.Number)

	_, err = k.AllocateBusinessSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT)
	require.ErrorIs(t, err, types.ErrAccountTypeAlreadyExists)
	require.Equal(t, uint32(2), k.GetNextBusinessSubaccountNumber(ctx, owner))

	fundingID, err := k.AllocateBusinessSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING)
	require.NoError(t, err)
	require.Equal(t, uint32(2), fundingID.Number)

	collisionOwner := owner + "1"
	k.SetSubaccount(ctx, types.Subaccount{
		Id: &types.SubaccountId{Owner: collisionOwner, Number: 1},
		AssetPositions: []*types.AssetPosition{
			testutil.CreateSingleAssetPosition(0, big.NewInt(1)),
		},
	})
	k.SetNextBusinessSubaccountNumber(ctx, collisionOwner, 1)

	thirdID, err := k.AllocateBusinessSubaccount(ctx, collisionOwner, types.AccountType_ACCOUNT_TYPE_SPOT)
	require.NoError(t, err)
	require.Equal(t, uint32(2), thirdID.Number)
}

func TestAllocateBusinessSubaccountRejectsInvalidAndExhaustedTypes(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	owner := "dydx1x2hd82qerp7lc0kf5cs3yekftupkrl620te6u2"

	_, err := k.AllocateBusinessSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_PERPETUAL)
	require.ErrorIs(t, err, types.ErrInvalidAccountType)

	k.SetNextBusinessSubaccountNumber(ctx, owner, types.MaxSubaccountIdNumber+1)
	_, err = k.AllocateBusinessSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT)
	require.ErrorIs(t, err, types.ErrSubaccountNumberExhausted)
}
