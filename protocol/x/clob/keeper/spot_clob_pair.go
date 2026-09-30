package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// CreateSpotClobPair creates a spot CLOB pair and stages its in-memory side effects.
func (k Keeper) CreateSpotClobPair(ctx sdk.Context, clobPair types.ClobPair) (types.ClobPair, error) {
	if clobPair.GetSpotClobMetadata() == nil {
		return types.ClobPair{}, errorsmod.Wrap(types.ErrInvalidClobPairParameter, "spot CLOB metadata is required")
	}
	if clobPair.Status == types.ClobPair_STATUS_FINAL_SETTLEMENT {
		return types.ClobPair{}, errorsmod.Wrapf(
			types.ErrInvalidClobPairParameter,
			"spot CLOB pair with id %d cannot be created in final settlement status",
			clobPair.Id,
		)
	}
	if err := k.ValidateClobPairCreation(ctx, &clobPair); err != nil {
		return types.ClobPair{}, err
	}
	k.SetClobPair(ctx, clobPair)
	if lib.IsDeliverTxMode(ctx) {
		if err := k.StageNewClobPairSideEffects(ctx, clobPair); err != nil {
			return clobPair, err
		}
	}
	return clobPair, nil
}

// CreateSpotClobPairAndMemStructs creates a spot CLOB pair during genesis initialization.
func (k Keeper) CreateSpotClobPairAndMemStructs(
	ctx sdk.Context,
	clobPair types.ClobPair,
) (types.ClobPair, error) {
	if clobPair.GetSpotClobMetadata() == nil {
		return types.ClobPair{}, errorsmod.Wrap(types.ErrInvalidClobPairParameter, "spot CLOB metadata is required")
	}
	if err := k.ValidateClobPairCreation(ctx, &clobPair); err != nil {
		return types.ClobPair{}, err
	}
	k.SetClobPair(ctx, clobPair)
	k.MemClob.CreateOrderbook(clobPair)
	return clobPair, nil
}
