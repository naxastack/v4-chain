package keeper_test

import (
	"math"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	authtest "github.com/dydxprotocol/v4-chain/protocol/testutil/auth"
	banktest "github.com/dydxprotocol/v4-chain/protocol/testutil/bank"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	testutil "github.com/dydxprotocol/v4-chain/protocol/testutil/util"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestCollateralPoolAssetQuantumsMatchesBankBalances(t *testing.T) {
	ctx, k, pricesKeeper, perpetualsKeeper, accountKeeper, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	keepertest.CreateTestMarkets(t, ctx, pricesKeeper)
	keepertest.CreateTestLiquidityTiers(t, ctx, perpetualsKeeper)
	keepertest.CreateTestPerpetuals(t, ctx, perpetualsKeeper)
	authtest.CreateTestModuleAccount(ctx, accountKeeper, types.ModuleName, []string{})

	owner := sampletest.AccAddress()
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	spotID := types.SubaccountId{Owner: owner, Number: 2}
	isolatedID := types.SubaccountId{Owner: owner, Number: 3}
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 300)
	setTransferTestSubaccount(k, ctx, spotID, types.AccountType_ACCOUNT_TYPE_SPOT, 200)
	k.SetSubaccount(ctx, types.Subaccount{
		Id:                 &isolatedID,
		AccountType:        types.AccountType_ACCOUNT_TYPE_PERPETUAL,
		AssetPositions:     testutil.CreateUsdcAssetPositions(big.NewInt(700)),
		PerpetualPositions: []*types.PerpetualPosition{&constants.PerpetualPosition_OneISOLong},
	})

	isolatedPool := authtypes.NewModuleAddress(
		types.ModuleName + ":" + lib.UintToString(constants.PerpetualPosition_OneISOLong.PerpetualId),
	)
	require.NoError(t, banktest.FundAccount(
		ctx,
		types.ModuleAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(500))),
		*bankKeeper,
	))
	require.NoError(t, banktest.FundAccount(
		ctx,
		isolatedPool,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(700))),
		*bankKeeper,
	))

	balances, err := k.GetCollateralPoolAssetQuantums(ctx, assettypes.AssetUsdc.Id)
	require.NoError(t, err)
	require.Equal(t, int64(500), balances[types.ModuleAddress.String()].Int64())
	require.Equal(t, int64(700), balances[isolatedPool.String()].Int64())
	require.Equal(
		t,
		bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.BigInt(),
		balances[types.ModuleAddress.String()],
	)
	require.Equal(
		t,
		bankKeeper.GetBalance(ctx, isolatedPool, constants.Usdc.Denom).Amount.BigInt(),
		balances[isolatedPool.String()],
	)
}

func TestFundingPathBankFailureLeavesPoolsAndLedgerUnchanged(t *testing.T) {
	ctx, k, pricesKeeper, perpetualsKeeper, accountKeeper, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))
	keepertest.CreateTestMarkets(t, ctx, pricesKeeper)
	keepertest.CreateTestLiquidityTiers(t, ctx, perpetualsKeeper)
	keepertest.CreateTestPerpetuals(t, ctx, perpetualsKeeper)
	authtest.CreateTestModuleAccount(ctx, accountKeeper, types.ModuleName, []string{})

	owner := sampletest.AccAddress()
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	perpetualID := types.SubaccountId{Owner: owner, Number: 2}
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 500)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)
	k.SetSubaccount(ctx, types.Subaccount{
		Id:                 &perpetualID,
		AccountType:        types.AccountType_ACCOUNT_TYPE_PERPETUAL,
		AssetPositions:     testutil.CreateUsdcAssetPositions(big.NewInt(600)),
		PerpetualPositions: []*types.PerpetualPosition{&constants.PerpetualPosition_OneISOLong},
	})

	isolatedPool := authtypes.NewModuleAddress(
		types.ModuleName + ":" + lib.UintToString(constants.PerpetualPosition_OneISOLong.PerpetualId),
	)
	require.NoError(t, banktest.FundAccount(
		ctx,
		types.ModuleAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(100))),
		*bankKeeper,
	))
	require.NoError(t, banktest.FundAccount(
		ctx,
		isolatedPool,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(600))),
		*bankKeeper,
	))
	eventsBefore := len(ctx.EventManager().Events())

	err := k.TransferFundsFromSubaccountToSubaccount(
		ctx,
		fundingID,
		perpetualID,
		assettypes.AssetUsdc.Id,
		big.NewInt(500),
	)
	require.ErrorIs(t, err, sdkerrors.ErrInsufficientFunds)
	require.Equal(t, int64(500), getTransferTestUsdcBalance(k, ctx, fundingID))
	require.Equal(t, int64(600), getTransferTestUsdcBalance(k, ctx, perpetualID))
	require.Equal(
		t,
		int64(100),
		bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64(),
	)
	require.Equal(
		t,
		int64(600),
		bankKeeper.GetBalance(ctx, isolatedPool, constants.Usdc.Denom).Amount.Int64(),
	)
	require.Len(t, ctx.EventManager().Events(), eventsBefore)
	require.Empty(t, keepertest.GetSubaccountUpdateEventsFromIndexerBlock(ctx, k))
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_FUNDING, k.GetSubaccount(ctx, fundingID).AccountType)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_PERPETUAL, k.GetSubaccount(ctx, perpetualID).AccountType)
}

func TestFundingPathPostLedgerFailureRollsBackStateAndEvents(t *testing.T) {
	ctx, k, _, _, _, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	spotID := types.SubaccountId{Owner: owner, Number: 1}
	fundingID := types.SubaccountId{Owner: owner, Number: 2}
	setTransferTestSubaccount(k, ctx, spotID, types.AccountType_ACCOUNT_TYPE_SPOT, 100)
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 0)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT, spotID)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)
	k.SetSpotOrderEpoch(ctx, spotID, assettypes.AssetUsdc.Id, math.MaxUint64)
	require.NoError(t, banktest.FundAccount(
		ctx,
		types.ModuleAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(100))),
		*bankKeeper,
	))
	eventsBefore := len(ctx.EventManager().Events())

	err := k.TransferFundsFromSubaccountToSubaccount(
		ctx,
		spotID,
		fundingID,
		assettypes.AssetUsdc.Id,
		big.NewInt(40),
	)
	require.ErrorIs(t, err, types.ErrSpotOrderEpochOverflow)
	require.Equal(t, int64(100), getTransferTestUsdcBalance(k, ctx, spotID))
	require.Zero(t, getTransferTestUsdcBalance(k, ctx, fundingID))
	require.Equal(
		t,
		int64(100),
		bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64(),
	)
	require.Equal(t, uint64(math.MaxUint64), k.GetSpotOrderEpoch(ctx, spotID, assettypes.AssetUsdc.Id))
	require.Len(t, ctx.EventManager().Events(), eventsBefore)
	require.Empty(t, keepertest.GetSubaccountUpdateEventsFromIndexerBlock(ctx, k))
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_SPOT, k.GetSubaccount(ctx, spotID).AccountType)
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_FUNDING, k.GetSubaccount(ctx, fundingID).AccountType)
}

func TestFundingDepositAuthorizationFailureDoesNotMutateState(t *testing.T) {
	ctx, k, _, _, _, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	otherAddress := sdk.MustAccAddressFromBech32(sampletest.AccAddress())
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 0)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)
	require.NoError(t, banktest.FundAccount(
		ctx,
		otherAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(100))),
		*bankKeeper,
	))
	eventsBefore := len(ctx.EventManager().Events())

	err := k.DepositFundsToFundingAccount(
		ctx,
		otherAddress,
		fundingID,
		assettypes.AssetUsdc.Id,
		big.NewInt(50),
	)
	require.ErrorIs(t, err, types.ErrFundingDepositOwnerMismatch)
	require.Zero(t, getTransferTestUsdcBalance(k, ctx, fundingID))
	require.Equal(
		t,
		int64(100),
		bankKeeper.GetBalance(ctx, otherAddress, constants.Usdc.Denom).Amount.Int64(),
	)
	require.Zero(t, bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64())
	require.Len(t, ctx.EventManager().Events(), eventsBefore)
	require.Empty(t, keepertest.GetSubaccountUpdateEventsFromIndexerBlock(ctx, k))
	require.Equal(t, types.AccountType_ACCOUNT_TYPE_FUNDING, k.GetSubaccount(ctx, fundingID).AccountType)
}
