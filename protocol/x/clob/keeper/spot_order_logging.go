package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	liblog "github.com/dydxprotocol/v4-chain/protocol/lib/log"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

const (
	spotLogComponent = "x/clob/spot"
	spotLogMarkerKey = "marker"
)

func addSpotOrderLogTags(ctx sdk.Context, orderId types.OrderId) sdk.Context {
	return liblog.AddPersistentTagsToLogger(
		ctx,
		liblog.Module, spotLogComponent,
		liblog.BlockHeight, ctx.BlockHeight(),
		liblog.OrderId, orderId,
		liblog.ClobPairId, orderId.GetClobPairId(),
		"subaccount_id", orderId.SubaccountId,
		"client_id", orderId.GetClientId(),
		"order_flags", orderId.GetOrderFlags(),
	)
}

func debugSpotOrder(ctx sdk.Context, orderId types.OrderId, marker string, keyvals ...interface{}) {
	ctx = addSpotOrderLogTags(ctx, orderId)
	liblog.DebugLog(ctx, "Spot order trace", append([]interface{}{spotLogMarkerKey, marker}, keyvals...)...)
}

func infoSpotOrder(ctx sdk.Context, orderId types.OrderId, marker string, keyvals ...interface{}) {
	ctx = addSpotOrderLogTags(ctx, orderId)
	liblog.InfoLog(ctx, "Spot order state changed", append([]interface{}{spotLogMarkerKey, marker}, keyvals...)...)
}
