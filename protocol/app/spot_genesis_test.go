package app_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	testapp "github.com/dydxprotocol/v4-chain/protocol/testutil/app"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	clobtypes "github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestValidateSpotGenesis(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()
	require.NoError(t, tApp.App.ValidateSpotGenesis(ctx))
}

func TestValidateSpotGenesisRejectsInconsistentActiveLifecycle(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()

	pairs := tApp.App.ClobKeeper.GetAllClobPairs(ctx)
	require.NotEmpty(t, pairs)
	corruptPair := pairs[0]
	corruptPair.Metadata = &clobtypes.ClobPair_SpotClobMetadata{
		SpotClobMetadata: &clobtypes.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0},
	}
	corruptPair.Status = clobtypes.ClobPair_STATUS_ACTIVE
	tApp.App.ClobKeeper.SetClobPair(ctx, corruptPair)

	tApp.App.ClobKeeper.SetSpotPairLifecycleState(
		ctx,
		corruptPair.GetClobPairId(),
		clobtypes.SpotPairLifecycleState{PauseSource: clobtypes.SpotPauseSourceManual},
	)
	require.ErrorContains(t, tApp.App.ValidateSpotGenesis(ctx), "inconsistent lifecycle state")
}

func TestValidateSpotGenesisRejectsMissingTypedAccountMapping(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()
	id := satypes.SubaccountId{Owner: sample.AccAddress(), Number: 1}
	tApp.App.SubaccountsKeeper.SetSubaccount(ctx, satypes.Subaccount{
		Id:          &id,
		AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
	})

	require.ErrorContains(t, tApp.App.ValidateSpotGenesis(ctx), "typed account mapping")
}

func TestValidateSpotGenesisRejectsReservationAggregateMismatch(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()
	id := satypes.SubaccountId{Owner: sample.AccAddress(), Number: 1}
	tApp.App.SubaccountsKeeper.SetSubaccount(ctx, satypes.Subaccount{
		Id:          &id,
		AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId:                  0,
			Quantums:                 dtypes.NewInt(100),
			StatefulReservedQuantums: dtypes.NewInt(10),
		}},
	})
	tApp.App.SubaccountsKeeper.SetTypedSubaccount(ctx, id.Owner, satypes.AccountType_ACCOUNT_TYPE_SPOT, id)

	require.ErrorContains(t, tApp.App.ValidateSpotGenesis(ctx), "reservation aggregate")
}

func TestValidateSpotGenesisRejectsEpochForNonSpotAccount(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()
	id := satypes.SubaccountId{Owner: sample.AccAddress(), Number: 1}
	tApp.App.SubaccountsKeeper.SetSubaccount(ctx, satypes.Subaccount{
		Id:          &id,
		AccountType: satypes.AccountType_ACCOUNT_TYPE_FUNDING,
	})
	tApp.App.SubaccountsKeeper.SetTypedSubaccount(ctx, id.Owner, satypes.AccountType_ACCOUNT_TYPE_FUNDING, id)
	tApp.App.SubaccountsKeeper.SetSpotOrderEpoch(ctx, id, 0, 1)

	require.ErrorContains(t, tApp.App.ValidateSpotGenesis(ctx), "epoch references non-spot subaccount")
}

func TestValidateSpotGenesisRejectsCollateralPoolMismatch(t *testing.T) {
	tApp := testapp.NewTestAppBuilder(t).Build()
	ctx := tApp.InitChain()
	pairs := tApp.App.ClobKeeper.GetAllClobPairs(ctx)
	require.NotEmpty(t, pairs)
	spotPair := pairs[0]
	spotPair.Metadata = &clobtypes.ClobPair_SpotClobMetadata{
		SpotClobMetadata: &clobtypes.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0},
	}
	spotPair.Status = clobtypes.ClobPair_STATUS_CANCEL_ONLY
	tApp.App.ClobKeeper.SetClobPair(ctx, spotPair)
	tApp.App.ClobKeeper.SetSpotPairLifecycleState(ctx, spotPair.GetClobPairId(), clobtypes.SpotPairLifecycleState{})
	id := satypes.SubaccountId{Owner: sample.AccAddress(), Number: 1}
	tApp.App.SubaccountsKeeper.SetSubaccount(ctx, satypes.Subaccount{
		Id:          &id,
		AccountType: satypes.AccountType_ACCOUNT_TYPE_FUNDING,
		AssetPositions: []*satypes.AssetPosition{{
			AssetId:                  0,
			Quantums:                 dtypes.NewInt(100),
			StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	tApp.App.SubaccountsKeeper.SetTypedSubaccount(ctx, id.Owner, satypes.AccountType_ACCOUNT_TYPE_FUNDING, id)

	require.ErrorContains(t, tApp.App.ValidateSpotGenesis(ctx), "collateral pool balance")
}
