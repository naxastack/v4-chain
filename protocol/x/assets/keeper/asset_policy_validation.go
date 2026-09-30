package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
)

func (k Keeper) getRequiredAssetPolicy(
	ctx sdk.Context,
	assetId uint32,
) (types.AssetPolicy, error) {
	if _, exists := k.GetAsset(ctx, assetId); !exists {
		return types.AssetPolicy{}, errorsmod.Wrapf(
			types.ErrAssetDoesNotExist,
			"asset id = %d",
			assetId,
		)
	}
	policy, exists := k.GetAssetPolicy(ctx, assetId)
	if !exists {
		return types.AssetPolicy{}, errorsmod.Wrapf(
			types.ErrAssetPolicyDoesNotExist,
			"asset id = %d",
			assetId,
		)
	}
	return policy, nil
}

// ValidateAssetDenom verifies the canonical denom for an asset id.
func (k Keeper) ValidateAssetDenom(
	ctx sdk.Context,
	assetId uint32,
	denom string,
) error {
	asset, exists := k.GetAsset(ctx, assetId)
	if !exists {
		return errorsmod.Wrapf(types.ErrAssetDoesNotExist, "asset id = %d", assetId)
	}
	if asset.Denom != denom {
		return errorsmod.Wrapf(
			types.ErrAssetDenomMismatch,
			"asset id = %d, expected denom = %s, actual denom = %s",
			assetId,
			asset.Denom,
			denom,
		)
	}
	return nil
}

// ValidateAssetForDeposit verifies that an asset accepts a new deposit.
func (k Keeper) ValidateAssetForDeposit(ctx sdk.Context, assetId uint32) error {
	policy, err := k.getRequiredAssetPolicy(ctx, assetId)
	if err != nil {
		return err
	}
	if !policy.AllowsDeposit() {
		return errorsmod.Wrapf(types.ErrAssetDepositsDisabled, "asset id = %d", assetId)
	}
	return nil
}

// ValidateAssetForWithdrawal verifies that an asset permits an exit.
func (k Keeper) ValidateAssetForWithdrawal(ctx sdk.Context, assetId uint32) error {
	policy, err := k.getRequiredAssetPolicy(ctx, assetId)
	if err != nil {
		return err
	}
	if !policy.AllowsWithdrawal() {
		return errorsmod.Wrapf(types.ErrAssetWithdrawalsDisabled, "asset id = %d", assetId)
	}
	return nil
}

// ValidateAssetForSpotTrading verifies that an asset is active for spot use.
func (k Keeper) ValidateAssetForSpotTrading(ctx sdk.Context, assetId uint32) error {
	policy, err := k.getRequiredAssetPolicy(ctx, assetId)
	if err != nil {
		return err
	}
	if !policy.AllowsSpotTrading() {
		return errorsmod.Wrapf(types.ErrAssetSpotTradingDisabled, "asset id = %d", assetId)
	}
	return nil
}

// ValidateAssetForPerpetual verifies that an asset is active for perpetual use.
func (k Keeper) ValidateAssetForPerpetual(ctx sdk.Context, assetId uint32) error {
	policy, err := k.getRequiredAssetPolicy(ctx, assetId)
	if err != nil {
		return err
	}
	if !policy.AllowsPerpetual() {
		return errorsmod.Wrapf(types.ErrAssetPerpetualDisabled, "asset id = %d", assetId)
	}
	return nil
}
