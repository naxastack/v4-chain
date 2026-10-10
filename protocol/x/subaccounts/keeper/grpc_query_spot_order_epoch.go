package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpotOrderEpoch(
	c context.Context,
	req *types.QuerySpotOrderEpochRequest,
) (*types.QuerySpotOrderEpochResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if _, err := sdk.AccAddressFromBech32(req.Owner); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner address")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	epoch := k.GetSpotOrderEpoch(ctx, types.SubaccountId{
		Owner:  req.Owner,
		Number: req.Number,
	}, req.AssetId)
	return &types.QuerySpotOrderEpochResponse{Epoch: epoch}, nil
}
