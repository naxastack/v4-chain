package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpotPairLifecycle(
	c context.Context,
	req *types.QuerySpotPairLifecycleRequest,
) (*types.QuerySpotPairLifecycleResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	state, found := k.GetSpotPairLifecycleState(ctx, types.ClobPairId(req.ClobPairId))
	if !found {
		return nil, status.Error(codes.NotFound, "spot pair lifecycle not found")
	}
	return &types.QuerySpotPairLifecycleResponse{
		ClobPairId:             req.ClobPairId,
		PauseSource:            types.SpotPairPauseSource(state.PauseSource),
		FinalSettlementStarted: state.FinalSettlementStarted,
	}, nil
}

func (k Keeper) SpotDelistCursor(
	c context.Context,
	req *types.QuerySpotDelistCursorRequest,
) (*types.QuerySpotDelistCursorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	cursor, found := k.GetSpotDelistCursor(ctx, req.ClobPairId)
	if !found {
		return nil, status.Error(codes.NotFound, "spot delist cursor not found")
	}
	return &types.QuerySpotDelistCursorResponse{Cursor: cursor}, nil
}

func (k Keeper) OrderFillAmount(
	c context.Context,
	req *types.QueryOrderFillAmountRequest,
) (*types.QueryOrderFillAmountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if _, err := sdk.AccAddressFromBech32(req.Owner); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner address")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	_, fillAmount, _ := k.GetOrderFillAmount(ctx, types.OrderId{
		SubaccountId: satypes.SubaccountId{
			Owner:  req.Owner,
			Number: req.Number,
		},
		ClientId:   req.ClientId,
		ClobPairId: req.ClobPairId,
		OrderFlags: req.OrderFlags,
	})
	return &types.QueryOrderFillAmountResponse{FillAmount: uint64(fillAmount)}, nil
}
