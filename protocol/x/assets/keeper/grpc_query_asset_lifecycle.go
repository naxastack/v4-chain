package keeper

import (
	"context"
	"fmt"

	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AssetPolicyState(
	goCtx context.Context,
	req *types.QueryAssetPolicyStateRequest,
) (*types.QueryAssetPolicyStateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	state, exists := k.GetAssetPolicyState(ctx, req.AssetId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("Asset policy state for asset id %d not found.", req.AssetId),
		)
	}
	return &types.QueryAssetPolicyStateResponse{State: state}, nil
}

func (k Keeper) AssetLifecycleTask(
	goCtx context.Context,
	req *types.QueryAssetLifecycleTaskRequest,
) (*types.QueryAssetLifecycleTaskResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	task, exists := k.GetAssetLifecycleTask(ctx, req.AssetId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("Asset lifecycle task for asset id %d not found.", req.AssetId),
		)
	}
	return &types.QueryAssetLifecycleTaskResponse{Task: task}, nil
}

func (k Keeper) AssetLifecycleSchedulerCursor(
	goCtx context.Context,
	req *types.QueryAssetLifecycleSchedulerCursorRequest,
) (*types.QueryAssetLifecycleSchedulerCursorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	return &types.QueryAssetLifecycleSchedulerCursorResponse{
		Cursor: k.GetAssetLifecycleSchedulerCursor(ctx),
	}, nil
}
