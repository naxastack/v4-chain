package keeper_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestReserveStatefulSpotQuantums(t *testing.T) {
	ctx, keeper, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	id := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
	keeper.SetSubaccount(ctx, types.Subaccount{
		Id:          &id,
		AccountType: types.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*types.AssetPosition{{
			AssetId:                  assettypes.AssetUsdc.Id,
			Quantums:                 dtypes.NewInt(100),
			StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})

	require.NoError(t, keeper.ReserveStatefulSpotQuantums(ctx, id, assettypes.AssetUsdc.Id, big.NewInt(40)))
	stored := keeper.GetSubaccount(ctx, id)
	require.Equal(t, int64(40), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, uint64(1), keeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))

	err := keeper.ReserveStatefulSpotQuantums(ctx, id, assettypes.AssetUsdc.Id, big.NewInt(61))
	require.ErrorIs(t, err, types.ErrStatefulReservedQuantumsInvalid)
	stored = keeper.GetSubaccount(ctx, id)
	require.Equal(t, int64(40), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, uint64(1), keeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))
}

func TestReserveStatefulSpotQuantumsEpochOverflowIsAtomic(t *testing.T) {
	ctx, keeper, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	id := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
	keeper.SetSubaccount(ctx, types.Subaccount{
		Id:          &id,
		AccountType: types.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*types.AssetPosition{{
			AssetId:                  assettypes.AssetUsdc.Id,
			Quantums:                 dtypes.NewInt(100),
			StatefulReservedQuantums: dtypes.ZeroInt(),
		}},
	})
	keeper.SetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id, math.MaxUint64)

	err := keeper.ReserveStatefulSpotQuantums(ctx, id, assettypes.AssetUsdc.Id, big.NewInt(40))
	require.ErrorIs(t, err, types.ErrSpotOrderEpochOverflow)
	stored := keeper.GetSubaccount(ctx, id)
	require.Zero(t, stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Sign())
	require.Equal(t, uint64(math.MaxUint64), keeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))
}
func TestReleaseStatefulSpotQuantums(t *testing.T) {
	ctx, keeper, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
	id := types.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
	keeper.SetSubaccount(ctx, types.Subaccount{
		Id:          &id,
		AccountType: types.AccountType_ACCOUNT_TYPE_SPOT,
		AssetPositions: []*types.AssetPosition{{
			AssetId:                  assettypes.AssetUsdc.Id,
			Quantums:                 dtypes.NewInt(100),
			StatefulReservedQuantums: dtypes.NewInt(40),
		}},
	})
	keeper.SetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id, 7)

	require.NoError(t, keeper.ReleaseStatefulSpotQuantums(ctx, id, assettypes.AssetUsdc.Id, big.NewInt(15)))
	stored := keeper.GetSubaccount(ctx, id)
	require.Equal(t, int64(25), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, uint64(7), keeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))

	err := keeper.ReleaseStatefulSpotQuantums(ctx, id, assettypes.AssetUsdc.Id, big.NewInt(26))
	require.ErrorIs(t, err, types.ErrStatefulReservedQuantumsInvalid)
	stored = keeper.GetSubaccount(ctx, id)
	require.Equal(t, int64(25), stored.AssetPositions[0].StatefulReservedQuantums.BigInt().Int64())
	require.Equal(t, uint64(7), keeper.GetSpotOrderEpoch(ctx, id, assettypes.AssetUsdc.Id))
}
