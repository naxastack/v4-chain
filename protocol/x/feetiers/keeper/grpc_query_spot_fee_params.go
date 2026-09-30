package keeper

import (
	"context"

	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpotFeeParams(
	c context.Context,
	req *types.QuerySpotFeeParamsRequest,
) (*types.QuerySpotFeeParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	return &types.QuerySpotFeeParamsResponse{Params: k.GetSpotFeeParams(ctx)}, nil
}
