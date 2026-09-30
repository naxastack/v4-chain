package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

func (k msgServer) UpdateSpotResourceParams(
	goCtx context.Context,
	msg *types.MsgUpdateSpotResourceParams,
) (*types.MsgUpdateSpotResourceParamsResponse, error) {
	if !k.Keeper.HasAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority %s", msg.Authority)
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	if err := k.Keeper.SetSpotResourceParams(ctx, msg.SpotResourceParams); err != nil {
		return nil, err
	}
	return &types.MsgUpdateSpotResourceParamsResponse{}, nil
}
