package keeper

import (
	"context"

	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpotResourceParams(
	c context.Context,
	req *types.QuerySpotResourceParamsRequest,
) (*types.QuerySpotResourceParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	return &types.QuerySpotResourceParamsResponse{Params: k.GetSpotResourceParams(ctx)}, nil
}
