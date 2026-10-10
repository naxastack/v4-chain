package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// GetSpotCandidateState returns the consensus-state inputs needed for local
// spot short-term reservation checks. It does not mutate consensus or MemCLOB state.
func (k Keeper) GetSpotCandidateState(
	ctx sdk.Context,
	order types.Order,
) (types.SpotCandidateState, error) {
	pair, found := k.GetClobPair(ctx, order.GetClobPairId())
	if !found {
		return types.SpotCandidateState{}, errorsmod.Wrapf(
			types.ErrInvalidClobPairParameter,
			"clob pair %d does not exist",
			order.GetClobPairId(),
		)
	}
	metadata := pair.GetSpotClobMetadata()
	if metadata == nil {
		return types.SpotCandidateState{}, errorsmod.Wrap(
			types.ErrInvalidClobPairParameter,
			"spot metadata is required",
		)
	}

	assetId := metadata.QuoteAssetId
	if !order.IsBuy() {
		assetId = metadata.BaseAssetId
	}
	total := new(big.Int)
	statefulReserved := new(big.Int)
	subaccount := k.subaccountsKeeper.GetSubaccount(ctx, order.OrderId.SubaccountId)
	for _, position := range subaccount.AssetPositions {
		if position.AssetId == assetId {
			total.Set(position.GetBigQuantums())
			statefulReserved.Set(position.StatefulReservedQuantums.BigInt())
			break
		}
	}

	return types.SpotCandidateState{
		ClobPair:                 pair,
		OutgoingAssetId:          assetId,
		TotalQuantums:            total,
		StatefulReservedQuantums: statefulReserved,
		CurrentOrderEpoch: k.subaccountsKeeper.GetSpotOrderEpoch(
			ctx,
			order.OrderId.SubaccountId,
			assetId,
		),
	}, nil
}
