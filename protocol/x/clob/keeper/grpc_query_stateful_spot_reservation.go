package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdkquery "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatefulSpotReservation queries one active spot reservation.
func (k Keeper) StatefulSpotReservation(
	c context.Context,
	req *types.QueryStatefulSpotReservationRequest,
) (*types.QueryStatefulSpotReservationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	reservation, found := k.GetStatefulSpotReservation(ctx, req.OrderId)
	if !found {
		return nil, status.Error(codes.NotFound, "spot reservation not found")
	}
	return &types.QueryStatefulSpotReservationResponse{Reservation: reservation}, nil
}

// SpotStatefulOrdersByClobPair queries active spot order ids in state-key order.
func (k Keeper) SpotStatefulOrdersByClobPair(
	c context.Context,
	req *types.QuerySpotStatefulOrdersByClobPairRequest,
) (*types.QuerySpotStatefulOrdersByClobPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := lib.UnwrapSDKContext(c, types.ModuleName)
	store := prefix.NewStore(
		ctx.KVStore(k.storeKey),
		spotStatefulOrdersByClobPairPrefix(req.ClobPairId),
	)
	orderIds := make([]types.OrderId, 0)
	page, err := sdkquery.Paginate(store, req.Pagination, func(_ []byte, value []byte) error {
		var orderId types.OrderId
		if err := orderId.Unmarshal(value); err != nil {
			return err
		}
		orderIds = append(orderIds, orderId)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QuerySpotStatefulOrdersByClobPairResponse{
		OrderIds:   orderIds,
		Pagination: page,
	}, nil
}
