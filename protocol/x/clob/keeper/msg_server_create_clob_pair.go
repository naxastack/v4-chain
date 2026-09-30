package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// CreateClobPair handles `MsgCreateClobPair`.
func (k msgServer) CreateClobPair(
	goCtx context.Context,
	msg *types.MsgCreateClobPair,
) (resp *types.MsgCreateClobPairResponse, err error) {
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)

	if !k.Keeper.HasAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(
			govtypes.ErrInvalidSigner,
			"invalid authority %s",
			msg.Authority,
		)
	}

	switch msg.ClobPair.Metadata.(type) {
	case *types.ClobPair_PerpetualClobMetadata:
		perpetualId, err := msg.ClobPair.GetPerpetualId()
		if err != nil {
			return nil, err
		}
		if _, err := k.Keeper.CreatePerpetualClobPair(
			ctx,
			msg.ClobPair.Id,
			perpetualId,
			satypes.BaseQuantums(msg.ClobPair.StepBaseQuantums),
			msg.ClobPair.QuantumConversionExponent,
			msg.ClobPair.SubticksPerTick,
			msg.ClobPair.Status,
		); err != nil {
			return nil, err
		}
	case *types.ClobPair_SpotClobMetadata:
		if _, err := k.Keeper.CreateSpotClobPair(ctx, msg.ClobPair); err != nil {
			return nil, err
		}
	default:
		return nil, errorsmod.Wrap(types.ErrInvalidClobPairParameter, "unsupported CLOB pair metadata")
	}
	return &types.MsgCreateClobPairResponse{}, nil
}
