package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateAsset registers an asset and its initial active policy atomically.
func (k msgServer) CreateAsset(
	goCtx context.Context,
	msg *types.MsgCreateAsset,
) (*types.MsgCreateAssetResponse, error) {
	if !k.HasAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority %s", msg.Authority)
	}
	if msg.Asset.Id != msg.Policy.AssetId {
		return nil, types.ErrAssetPolicyAssetIdMismatch
	}
	if msg.Policy.Status != types.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE {
		return nil, types.ErrInvalidAssetPolicyStatus
	}

	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	cacheCtx, write := ctx.CacheContext()

	asset := msg.Asset
	if _, err := k.Keeper.CreateAsset(
		cacheCtx,
		asset.Id,
		asset.Symbol,
		asset.Denom,
		asset.DenomExponent,
		asset.HasMarket,
		asset.MarketId,
		asset.AtomicResolution,
	); err != nil {
		return nil, err
	}
	if err := k.Keeper.CreateAssetPolicy(cacheCtx, msg.Policy); err != nil {
		return nil, err
	}

	write()
	return &types.MsgCreateAssetResponse{}, nil
}

// UpdateAssetPolicy updates the mutable policy for an existing asset.
func (k msgServer) UpdateAssetPolicy(
	goCtx context.Context,
	msg *types.MsgUpdateAssetPolicy,
) (*types.MsgUpdateAssetPolicyResponse, error) {
	if !k.HasAuthority(msg.Authority) {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority %s", msg.Authority)
	}
	ctx := lib.UnwrapSDKContext(goCtx, types.ModuleName)
	if err := k.Keeper.UpdateAssetPolicy(ctx, msg.Policy); err != nil {
		return nil, err
	}
	return &types.MsgUpdateAssetPolicyResponse{}, nil
}
