package keeper_test

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktest "github.com/dydxprotocol/v4-chain/protocol/testutil/bank"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestFundingAccountDepositAndWithdrawal(t *testing.T) {
	ctx, k, _, _, _, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	ownerAddress := sdk.MustAccAddressFromBech32(owner)
	recipientAddress := sdk.MustAccAddressFromBech32(sampletest.AccAddress())
	require.NoError(t, banktest.FundAccount(
		ctx,
		ownerAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(1_000))),
		*bankKeeper,
	))

	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	k.SetSubaccount(ctx, types.Subaccount{
		Id:          &fundingID,
		AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING,
	})
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)

	require.NoError(t, k.DepositFundsToFundingAccount(
		ctx,
		ownerAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(500),
	))
	fundingSubaccount := k.GetSubaccount(ctx, fundingID)
	require.Equal(t, int64(500), fundingSubaccount.GetUsdcPosition().Int64())
	require.Equal(t, int64(500), bankKeeper.GetBalance(ctx, ownerAddress, constants.Usdc.Denom).Amount.Int64())
	require.Equal(t, int64(500), bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64())

	require.NoError(t, k.WithdrawFundsFromFundingAccount(
		ctx,
		fundingID,
		recipientAddress,
		constants.Usdc.Id,
		big.NewInt(200),
	))
	fundingSubaccount = k.GetSubaccount(ctx, fundingID)
	require.Equal(t, int64(300), fundingSubaccount.GetUsdcPosition().Int64())
	require.Equal(t, int64(300), bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64())
	require.Equal(t, int64(200), bankKeeper.GetBalance(ctx, recipientAddress, constants.Usdc.Denom).Amount.Int64())
}

func TestFundingAccountDepositValidationAndRollback(t *testing.T) {
	ctx, k, _, _, _, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	ownerAddress := sdk.MustAccAddressFromBech32(owner)
	require.NoError(t, banktest.FundAccount(
		ctx,
		ownerAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(100))),
		*bankKeeper,
	))
	fundingID := types.SubaccountId{Owner: owner, Number: 1}

	err := k.DepositFundsToFundingAccount(
		ctx,
		ownerAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(10),
	)
	require.ErrorIs(t, err, types.ErrFundingSubaccountNotFound)

	k.SetSubaccount(ctx, types.Subaccount{Id: &fundingID, AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING})
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)

	otherAddress := sdk.MustAccAddressFromBech32(sampletest.AccAddress())
	err = k.DepositFundsToFundingAccount(
		ctx,
		otherAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(10),
	)
	require.ErrorIs(t, err, types.ErrFundingDepositOwnerMismatch)

	err = k.DepositFundsToFundingAccount(
		ctx,
		ownerAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(200),
	)
	require.Error(t, err)
	fundingSubaccount := k.GetSubaccount(ctx, fundingID)
	require.Zero(t, fundingSubaccount.GetUsdcPosition().Sign())
	require.Equal(t, int64(100), bankKeeper.GetBalance(ctx, ownerAddress, constants.Usdc.Denom).Amount.Int64())
	require.Zero(t, bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64())
}

func TestFundingWithdrawalAllowedWhenDepositsPaused(t *testing.T) {
	ctx, k, _, _, _, bankKeeper, assetsKeeper, _, _, _, _ :=
		keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	ownerAddress := sdk.MustAccAddressFromBech32(owner)
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	k.SetSubaccount(ctx, types.Subaccount{Id: &fundingID, AccountType: types.AccountType_ACCOUNT_TYPE_FUNDING})
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, fundingID)
	require.NoError(t, banktest.FundAccount(
		ctx,
		ownerAddress,
		sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(100))),
		*bankKeeper,
	))
	require.NoError(t, k.DepositFundsToFundingAccount(
		ctx,
		ownerAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(100),
	))

	policy := assettypes.AssetPolicyUsdc
	policy.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	policy.DepositsEnabled = false
	policy.SpotTradingEnabled = false
	require.NoError(t, assetsKeeper.UpdateAssetPolicy(ctx, policy))

	err := k.DepositFundsToFundingAccount(
		ctx,
		ownerAddress,
		fundingID,
		constants.Usdc.Id,
		big.NewInt(1),
	)
	require.ErrorIs(t, err, assettypes.ErrAssetDepositsDisabled)

	require.NoError(t, k.WithdrawFundsFromFundingAccount(
		ctx,
		fundingID,
		ownerAddress,
		constants.Usdc.Id,
		big.NewInt(100),
	))
	fundingSubaccount := k.GetSubaccount(ctx, fundingID)
	require.Zero(t, fundingSubaccount.GetUsdcPosition().Sign())
	require.Equal(t, int64(100), bankKeeper.GetBalance(ctx, ownerAddress, constants.Usdc.Denom).Amount.Int64())
}
