package keeper_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"github.com/stretchr/testify/require"
)

func TestValidateSubaccountEquityTierLimitForShortTermOrder_ResultHeldUntilCleared(t *testing.T) {
	ks, ctx := setupShortTermOrderKeeper(t, constants.Alice_Num0_10_000USD, []types.EquityTierLimit{
		{UsdTncRequired: dtypes.NewInt(0), Limit: 0},
		{UsdTncRequired: dtypes.NewInt(70_000_000_000), Limit: 1000}, // $70,000
	})
	order := restingBuyOrder(constants.Alice_Num0, 0)

	err := ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order)
	require.ErrorIs(t, err, types.ErrOrderWouldExceedMaxOpenOrdersEquityTierLimit)

	ks.SubaccountsKeeper.SetSubaccount(ctx, constants.Alice_Num0_100_000USD)
	err = ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order)
	require.ErrorIs(t, err, types.ErrOrderWouldExceedMaxOpenOrdersEquityTierLimit)

	ks.ClobKeeper.ClearShortTermEquityTierResults()
	require.NoError(t, ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order))

	ks.SubaccountsKeeper.SetSubaccount(ctx, constants.Alice_Num0_10_000USD)
	require.NoError(t, ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order))

	ks.ClobKeeper.ClearShortTermEquityTierResults()
	err = ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order)
	require.ErrorIs(t, err, types.ErrOrderWouldExceedMaxOpenOrdersEquityTierLimit)
}

func TestValidateSubaccountEquityTierLimitForShortTermOrder_SpotPairSkipsPerpetualFloor(t *testing.T) {
	ks, ctx := setupShortTermOrderKeeper(t, constants.Alice_Num0_10_000USD, []types.EquityTierLimit{
		{UsdTncRequired: dtypes.NewInt(0), Limit: 0},
		{UsdTncRequired: dtypes.NewInt(70_000_000_000), Limit: 1000},
	})
	_, err := ks.AssetsKeeper.CreateAsset(ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicy{
			AssetId:            assetId,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			DepositsEnabled:    true,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
	}
	_, err = ks.ClobKeeper.CreateSpotClobPairAndMemStructs(ctx, spotPairForIndex(100, 1))
	require.NoError(t, err)

	order := restingBuyOrder(constants.Alice_Num0, 1)
	order.OrderId.ClobPairId = 100
	order.MaxTradingFeePpm = 1000
	require.NoError(t, ks.ClobKeeper.ValidateSubaccountEquityTierLimitForShortTermOrder(ctx, order))
}
