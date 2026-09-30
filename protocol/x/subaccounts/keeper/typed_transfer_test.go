package keeper_test

import (
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
	subaccountskeeper "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func setTransferTestSubaccount(
	k *subaccountskeeper.Keeper,
	ctx sdk.Context,
	id types.SubaccountId,
	accountType types.AccountType,
	quantums int64,
) {
	subaccount := types.Subaccount{Id: &id, AccountType: accountType}
	if quantums != 0 {
		subaccount.AssetPositions = testutil.CreateUsdcAssetPositions(big.NewInt(quantums))
	}
	k.SetSubaccount(ctx, subaccount)
	if accountType.IsBusinessAccountType() {
		k.SetTypedSubaccount(ctx, id.Owner, accountType, id)
	}
}

func getTransferTestUsdcBalance(
	k *subaccountskeeper.Keeper,
	ctx sdk.Context,
	id types.SubaccountId,
) int64 {
	subaccount := k.GetSubaccount(ctx, id)
	return subaccount.GetUsdcPosition().Int64()
}

func TestTypedTransferRouteMatrix(t *testing.T) {
	tests := []struct {
		name          string
		senderType    types.AccountType
		recipientType types.AccountType
		wantErr       error
	}{
		{
			name:          "funding to spot",
			senderType:    types.AccountType_ACCOUNT_TYPE_FUNDING,
			recipientType: types.AccountType_ACCOUNT_TYPE_SPOT,
		},
		{
			name:          "spot to funding",
			senderType:    types.AccountType_ACCOUNT_TYPE_SPOT,
			recipientType: types.AccountType_ACCOUNT_TYPE_FUNDING,
		},
		{
			name:          "funding to perpetual",
			senderType:    types.AccountType_ACCOUNT_TYPE_FUNDING,
			recipientType: types.AccountType_ACCOUNT_TYPE_PERPETUAL,
		},
		{
			name:          "perpetual to funding",
			senderType:    types.AccountType_ACCOUNT_TYPE_PERPETUAL,
			recipientType: types.AccountType_ACCOUNT_TYPE_FUNDING,
		},
		{
			name:          "spot to perpetual is rejected",
			senderType:    types.AccountType_ACCOUNT_TYPE_SPOT,
			recipientType: types.AccountType_ACCOUNT_TYPE_PERPETUAL,
			wantErr:       types.ErrTypedTransferRouteNotAllowed,
		},
		{
			name:          "perpetual to spot is rejected",
			senderType:    types.AccountType_ACCOUNT_TYPE_PERPETUAL,
			recipientType: types.AccountType_ACCOUNT_TYPE_SPOT,
			wantErr:       types.ErrTypedTransferRouteNotAllowed,
		},
		{
			name:          "spot to spot is rejected",
			senderType:    types.AccountType_ACCOUNT_TYPE_SPOT,
			recipientType: types.AccountType_ACCOUNT_TYPE_SPOT,
			wantErr:       types.ErrTypedTransferRouteNotAllowed,
		},
		{
			name:          "funding to funding is rejected",
			senderType:    types.AccountType_ACCOUNT_TYPE_FUNDING,
			recipientType: types.AccountType_ACCOUNT_TYPE_FUNDING,
			wantErr:       types.ErrTypedTransferRouteNotAllowed,
		},
		{
			name:          "business transfer across owners is rejected",
			senderType:    types.AccountType_ACCOUNT_TYPE_FUNDING,
			recipientType: types.AccountType_ACCOUNT_TYPE_SPOT,
			wantErr:       types.ErrTypedTransferOwnerMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, k, _, _, _, _, assetsKeeper, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
			require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
			require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))
			senderOwner := sampletest.AccAddress()
			recipientOwner := senderOwner
			if tt.wantErr == types.ErrTypedTransferOwnerMismatch {
				recipientOwner = sampletest.AccAddress()
			}
			senderID := types.SubaccountId{Owner: senderOwner, Number: 1}
			recipientID := types.SubaccountId{Owner: recipientOwner, Number: 2}
			setTransferTestSubaccount(k, ctx, senderID, tt.senderType, 100)
			setTransferTestSubaccount(k, ctx, recipientID, tt.recipientType, 1)

			err := k.TransferFundsFromSubaccountToSubaccount(
				ctx,
				senderID,
				recipientID,
				assettypes.AssetUsdc.Id,
				big.NewInt(25),
			)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, int64(100), getTransferTestUsdcBalance(k, ctx, senderID))
				require.Equal(t, int64(1), getTransferTestUsdcBalance(k, ctx, recipientID))
				return
			}
			require.NoError(t, err)
			require.Equal(t, int64(75), getTransferTestUsdcBalance(k, ctx, senderID))
			require.Equal(t, int64(26), getTransferTestUsdcBalance(k, ctx, recipientID))
		})
	}
}

func TestTypedTransferRequiresPersistedCanonicalAccounts(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	owner := sampletest.AccAddress()
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	spotID := types.SubaccountId{Owner: owner, Number: 2}
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 100)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_SPOT, spotID)

	err := k.TransferFundsFromSubaccountToSubaccount(
		ctx, fundingID, spotID, assettypes.AssetUsdc.Id, big.NewInt(25),
	)
	require.ErrorIs(t, err, types.ErrTypedTransferSubaccountNotFound)
	require.Equal(t, int64(100), getTransferTestUsdcBalance(k, ctx, fundingID))

	setTransferTestSubaccount(k, ctx, spotID, types.AccountType_ACCOUNT_TYPE_SPOT, 1)
	k.SetTypedSubaccount(ctx, owner, types.AccountType_ACCOUNT_TYPE_FUNDING, spotID)
	err = k.TransferFundsFromSubaccountToSubaccount(
		ctx, fundingID, spotID, assettypes.AssetUsdc.Id, big.NewInt(25),
	)
	require.ErrorIs(t, err, types.ErrTypedTransferMappingInvalid)
	require.Equal(t, int64(100), getTransferTestUsdcBalance(k, ctx, fundingID))
	require.Equal(t, int64(1), getTransferTestUsdcBalance(k, ctx, spotID))
}

func TestTypedTransferAssetPolicy(t *testing.T) {
	ctx, k, _, _, _, _, assetsKeeper, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))
	owner := sampletest.AccAddress()
	fundingID := types.SubaccountId{Owner: owner, Number: 1}
	spotID := types.SubaccountId{Owner: owner, Number: 2}
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 1)
	setTransferTestSubaccount(k, ctx, spotID, types.AccountType_ACCOUNT_TYPE_SPOT, 100)

	policy, exists := assetsKeeper.GetAssetPolicy(ctx, assettypes.AssetUsdc.Id)
	require.True(t, exists)
	policy.Status = assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED
	require.NoError(t, assetsKeeper.UpdateAssetPolicy(ctx, policy))

	err := k.TransferFundsFromSubaccountToSubaccount(
		ctx, fundingID, spotID, assettypes.AssetUsdc.Id, big.NewInt(1),
	)
	require.ErrorIs(t, err, assettypes.ErrAssetSpotTradingDisabled)

	err = k.TransferFundsFromSubaccountToSubaccount(
		ctx, spotID, fundingID, assettypes.AssetUsdc.Id, big.NewInt(25),
	)
	require.NoError(t, err)
	require.Equal(t, int64(75), getTransferTestUsdcBalance(k, ctx, spotID))
	require.Equal(t, int64(26), getTransferTestUsdcBalance(k, ctx, fundingID))
}

func TestTypedTransferAcrossCollateralPoolsIsAtomic(t *testing.T) {
	tests := []struct {
		name                     string
		senderPoolBalance        int64
		wantErr                  error
		wantSenderBalance        int64
		wantRecipientBalance     int64
		wantSenderPoolBalance    int64
		wantRecipientPoolBalance int64
	}{
		{
			name:                     "funding to isolated perpetual moves bank collateral",
			senderPoolBalance:        600,
			wantSenderBalance:        0,
			wantRecipientBalance:     1100,
			wantSenderPoolBalance:    100,
			wantRecipientPoolBalance: 1200,
		},
		{
			name:                     "bank failure rolls back typed ledger updates",
			senderPoolBalance:        100,
			wantErr:                  sdkerrors.ErrInsufficientFunds,
			wantSenderBalance:        500,
			wantRecipientBalance:     600,
			wantSenderPoolBalance:    100,
			wantRecipientPoolBalance: 700,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			perpetual := types.Subaccount{
				Id:                 &perpetualID,
				AccountType:        types.AccountType_ACCOUNT_TYPE_PERPETUAL,
				AssetPositions:     testutil.CreateUsdcAssetPositions(big.NewInt(600)),
				PerpetualPositions: []*types.PerpetualPosition{&constants.PerpetualPosition_OneISOLong},
			}
			k.SetSubaccount(ctx, perpetual)

			isolatedPool := authtypes.NewModuleAddress(
				types.ModuleName + ":" + lib.UintToString(constants.PerpetualPosition_OneISOLong.PerpetualId),
			)
			require.NoError(t, banktest.FundAccount(
				ctx,
				types.ModuleAddress,
				sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(tt.senderPoolBalance))),
				*bankKeeper,
			))
			require.NoError(t, banktest.FundAccount(
				ctx,
				isolatedPool,
				sdk.NewCoins(sdk.NewCoin(constants.Usdc.Denom, sdkmath.NewInt(700))),
				*bankKeeper,
			))

			err := k.TransferFundsFromSubaccountToSubaccount(
				ctx, fundingID, perpetualID, assettypes.AssetUsdc.Id, big.NewInt(500),
			)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wantSenderBalance, getTransferTestUsdcBalance(k, ctx, fundingID))
			require.Equal(t, tt.wantRecipientBalance, getTransferTestUsdcBalance(k, ctx, perpetualID))
			require.Equal(
				t,
				tt.wantSenderPoolBalance,
				bankKeeper.GetBalance(ctx, types.ModuleAddress, constants.Usdc.Denom).Amount.Int64(),
			)
			require.Equal(
				t,
				tt.wantRecipientPoolBalance,
				bankKeeper.GetBalance(ctx, isolatedPool, constants.Usdc.Denom).Amount.Int64(),
			)
		})
	}
}
