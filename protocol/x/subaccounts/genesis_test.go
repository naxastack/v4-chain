package subaccounts_test

import (
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/nullify"
	testutil "github.com/dydxprotocol/v4-chain/protocol/testutil/util"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Subaccounts: []types.Subaccount{
			{
				Id: &types.SubaccountId{
					Owner:  "foo",
					Number: uint32(0),
				},
				AssetPositions: testutil.CreateUsdcAssetPositions(big.NewInt(1_000)),
			},
			{
				Id: &types.SubaccountId{
					Owner:  "bar",
					Number: uint32(99),
				},
				AssetPositions: testutil.CreateUsdcAssetPositions(big.NewInt(1_000)),
			},
		},
	}

	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	subaccounts.InitGenesis(ctx, *k, genesisState)
	assertSubaccountUpdateEventsInIndexerBlock(t, k, ctx, 2)
	got := subaccounts.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState) //nolint:staticcheck
	nullify.Fill(got)           //nolint:staticcheck

	require.ElementsMatch(t, genesisState.Subaccounts, got.Subaccounts)
}

// assertSubaccountUpdateEventsInIndexerBlock checks that the number of subaccount update events
// included in the Indexer block kafka message.
func assertSubaccountUpdateEventsInIndexerBlock(
	t *testing.T,
	k *keeper.Keeper,
	ctx sdk.Context,
	numSubaccounts int,
) {
	subaccountUpdates := keepertest.GetSubaccountUpdateEventsFromIndexerBlock(ctx, k)
	require.Len(t, subaccountUpdates, numSubaccounts)
}

func TestGenesisPreservesTypedMappingsAndBusinessCursors(t *testing.T) {
	spotOwner := "dydx1spotowner0000000000000000000000000000000"
	fundingOwner := "dydx1fundingowner0000000000000000000000000000"
	genesisState := types.GenesisState{
		Subaccounts: []types.Subaccount{
			{
				Id:             &types.SubaccountId{Owner: spotOwner, Number: 1},
				AccountType:    types.AccountType_ACCOUNT_TYPE_SPOT,
				AssetPositions: testutil.CreateUsdcAssetPositions(big.NewInt(1_000)),
			},
			{
				Id:             &types.SubaccountId{Owner: fundingOwner, Number: 1},
				AccountType:    types.AccountType_ACCOUNT_TYPE_FUNDING,
				AssetPositions: testutil.CreateUsdcAssetPositions(big.NewInt(2_000)),
			},
		},
		TypedSubaccounts: []types.TypedSubaccount{
			{Owner: spotOwner, AccountType: types.AccountType_ACCOUNT_TYPE_SPOT, SubaccountId: &types.SubaccountId{Owner: spotOwner, Number: 1}},
			{Owner: fundingOwner, AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING, SubaccountId: &types.SubaccountId{Owner: fundingOwner, Number: 1}},
		},
		BusinessNumberCursors: []types.BusinessSubaccountNumberCursor{
			{Owner: spotOwner, NextNumber: 2},
			{Owner: fundingOwner, NextNumber: 2},
		},
	}

	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	subaccounts.InitGenesis(ctx, *k, genesisState)
	got := subaccounts.ExportGenesis(ctx, *k)
	require.NotNil(t, got)
	require.ElementsMatch(t, genesisState.Subaccounts, got.Subaccounts)
	require.ElementsMatch(t, genesisState.TypedSubaccounts, got.TypedSubaccounts)
	require.ElementsMatch(t, genesisState.BusinessNumberCursors, got.BusinessNumberCursors)
}
