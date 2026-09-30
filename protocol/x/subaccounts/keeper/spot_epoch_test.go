package keeper_test

import (
	"math"
	"math/big"
	"testing"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestSpotOrderEpochPersistenceAndOverflow(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	id := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}

	require.Zero(t, k.GetSpotOrderEpoch(ctx, id, 0))
	epoch, err := k.IncrementSpotOrderEpoch(ctx, id, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(1), epoch)
	require.Equal(t, uint64(1), k.GetSpotOrderEpoch(ctx, id, 0))

	k.SetSpotOrderEpoch(ctx, id, 0, math.MaxUint64)
	_, err = k.IncrementSpotOrderEpoch(ctx, id, 0)
	require.ErrorIs(t, err, types.ErrSpotOrderEpochOverflow)
	require.Equal(t, uint64(math.MaxUint64), k.GetSpotOrderEpoch(ctx, id, 0))
}

func TestSpotTransferOutIncrementsOrderEpochAtomically(t *testing.T) {
	ctx, k, _, _, _, _, assetsKeeper, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	require.NoError(t, keepertest.CreateUsdcAsset(ctx, assetsKeeper))
	require.NoError(t, assetsKeeper.CreateAssetPolicy(ctx, assettypes.AssetPolicyUsdc))

	owner := sampletest.AccAddress()
	spotID := types.SubaccountId{Owner: owner, Number: 1}
	fundingID := types.SubaccountId{Owner: owner, Number: 2}
	setTransferTestSubaccount(k, ctx, spotID, types.AccountType_ACCOUNT_TYPE_SPOT, 100)
	setTransferTestSubaccount(k, ctx, fundingID, types.AccountType_ACCOUNT_TYPE_FUNDING, 0)

	require.NoError(t, k.TransferFundsFromSubaccountToSubaccount(
		ctx,
		spotID,
		fundingID,
		assettypes.AssetUsdc.Id,
		big.NewInt(25),
	))
	require.Equal(t, uint64(1), k.GetSpotOrderEpoch(ctx, spotID, assettypes.AssetUsdc.Id))

	err := k.TransferFundsFromSubaccountToSubaccount(
		ctx,
		spotID,
		fundingID,
		assettypes.AssetUsdc.Id,
		big.NewInt(1_000),
	)
	require.Error(t, err)
	require.Equal(t, uint64(1), k.GetSpotOrderEpoch(ctx, spotID, assettypes.AssetUsdc.Id))

	require.NoError(t, k.TransferFundsFromSubaccountToSubaccount(
		ctx,
		fundingID,
		spotID,
		assettypes.AssetUsdc.Id,
		big.NewInt(10),
	))
	require.Equal(t, uint64(1), k.GetSpotOrderEpoch(ctx, spotID, assettypes.AssetUsdc.Id))
}

func TestGetAllSpotOrderEpochsDeterministic(t *testing.T) {
	ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	first := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 2}
	second := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}

	k.SetSpotOrderEpoch(ctx, second, 2, 3)
	k.SetSpotOrderEpoch(ctx, first, 1, 4)
	k.SetSpotOrderEpoch(ctx, first, 0, 2)

	epochs := k.GetAllSpotOrderEpochs(ctx)
	require.Len(t, epochs, 3)
	for i := 1; i < len(epochs); i++ {
		previous := epochs[i-1]
		current := epochs[i]
		require.True(t,
			previous.SubaccountId.Owner < current.SubaccountId.Owner ||
				(previous.SubaccountId.Owner == current.SubaccountId.Owner &&
					(previous.SubaccountId.Number < current.SubaccountId.Number ||
						(previous.SubaccountId.Number == current.SubaccountId.Number &&
							previous.AssetId < current.AssetId))),
		)
	}
}
