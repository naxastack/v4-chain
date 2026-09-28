package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns the subaccounts message service implementation.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return msgServer{Keeper: keeper}
}

func (s msgServer) CreateSubaccount(
	goCtx context.Context,
	msg *types.MsgCreateSubaccount,
) (*types.MsgCreateSubaccountResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	cacheCtx, write := ctx.CacheContext()
	id, err := s.AllocateBusinessSubaccount(cacheCtx, msg.Owner, msg.AccountType)
	if err != nil {
		return nil, err
	}

	s.SetSubaccount(cacheCtx, types.Subaccount{
		Id:          &id,
		AccountType: msg.AccountType,
	})
	write()

	return &types.MsgCreateSubaccountResponse{SubaccountId: &id}, nil
}
