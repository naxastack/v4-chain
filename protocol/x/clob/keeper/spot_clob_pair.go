package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

// ValidateSpotMarketActivation validates all module-local prerequisites for a
// Spot market to enter or remain ACTIVE. Cross-module account and custody
// invariants are validated by the app-level genesis validator after all
// modules have initialized.
func (k Keeper) ValidateSpotMarketActivation(ctx sdk.Context, clobPair types.ClobPair) error {
	if clobPair.GetSpotClobMetadata() == nil {
		return errorsmod.Wrap(types.ErrInvalidClobPairParameter, "spot CLOB metadata is required")
	}
	if clobPair.Status != types.ClobPair_STATUS_ACTIVE {
		return errorsmod.Wrapf(
			types.ErrInvalidClobPairStatusTransition,
			"spot CLOB pair %d must be ACTIVE for activation validation",
			clobPair.Id,
		)
	}
	if err := k.validateClobPair(ctx, &clobPair); err != nil {
		return err
	}
	if err := k.GetSpotResourceParams(ctx).Validate(); err != nil {
		return errorsmod.Wrap(err, "invalid spot resource params")
	}
	if err := k.feeTiersKeeper.GetSpotFeeParams(ctx).Validate(); err != nil {
		return errorsmod.Wrap(err, "invalid spot fee params")
	}
	return nil
}

// CreateSpotClobPair creates a spot CLOB pair and stages its in-memory side effects.
func (k Keeper) CreateSpotClobPair(ctx sdk.Context, clobPair types.ClobPair) (types.ClobPair, error) {
	if clobPair.GetSpotClobMetadata() == nil {
		return types.ClobPair{}, errorsmod.Wrap(types.ErrInvalidClobPairParameter, "spot CLOB metadata is required")
	}
	if clobPair.Status != types.ClobPair_STATUS_CANCEL_ONLY {
		return types.ClobPair{}, errorsmod.Wrapf(
			types.ErrInvalidClobPairParameter,
			"spot CLOB pair with id %d must be created in cancel-only status",
			clobPair.Id,
		)
	}
	if err := k.ValidateClobPairCreation(ctx, &clobPair); err != nil {
		return types.ClobPair{}, err
	}
	k.SetClobPair(ctx, clobPair)
	k.SetSpotPairLifecycleState(ctx, clobPair.GetClobPairId(), types.SpotPairLifecycleState{})
	k.indexSpotPairByAssets(ctx, clobPair)
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
	k.SetSpotPairLifecycleState(ctx, clobPair.GetClobPairId(), types.SpotPairLifecycleState{})
	k.indexSpotPairByAssets(ctx, clobPair)
	k.MemClob.CreateOrderbook(clobPair)
	return clobPair, nil
}
