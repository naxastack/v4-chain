package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssetPolicy queries the policy for one asset.
func (k Keeper) AssetPolicy(
	goCtx context.Context,
	req *types.QueryAssetPolicyRequest,
) (*types.QueryAssetPolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	policy, exists := k.GetAssetPolicy(ctx, req.AssetId)
	if !exists {
		return nil, status.Error(
			codes.NotFound,
			fmt.Sprintf("Asset policy for asset id %d not found.", req.AssetId),
		)
	}
	return &types.QueryAssetPolicyResponse{Policy: policy}, nil
}

// AllAssetPolicies queries asset policies with pagination.
func (k Keeper) AllAssetPolicies(
	goCtx context.Context,
	req *types.QueryAllAssetPoliciesRequest,
) (*types.QueryAllAssetPoliciesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.AssetPolicyKeyPrefix))
	policies := make([]types.AssetPolicy, 0)

	pageRes, err := query.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		var policy types.AssetPolicy
		if err := k.cdc.Unmarshal(value, &policy); err != nil {
			return err
		}
		policies = append(policies, policy)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryAllAssetPoliciesResponse{
		Policies:   policies,
		Pagination: pageRes,
	}, nil
}
