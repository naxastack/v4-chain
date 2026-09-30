package keeper

import (
	errorsmod "cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

func (k Keeper) hasSubaccount(ctx sdk.Context, id types.SubaccountId) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SubaccountKeyPrefix))
	return store.Has(id.ToStateKey())
}

// validateTypedTransfer validates business-account routing and returns whether
// the transfer uses the typed-account path. Transfers between perpetual or
// legacy unspecified accounts retain the existing behavior.
func (k Keeper) validateTypedTransfer(
	ctx sdk.Context,
	senderID types.SubaccountId,
	recipientID types.SubaccountId,
	assetID uint32,
) (bool, error) {
	sender := k.GetSubaccount(ctx, senderID)
	recipient := k.GetSubaccount(ctx, recipientID)
	senderType := k.getTransferAccountType(ctx, senderID, sender.AccountType)
	recipientType := k.getTransferAccountType(ctx, recipientID, recipient.AccountType)
	senderIsBusiness := senderType.IsBusinessAccountType()
	recipientIsBusiness := recipientType.IsBusinessAccountType()
	if !senderIsBusiness && !recipientIsBusiness {
		return false, nil
	}

	if !k.hasSubaccount(ctx, senderID) || !k.hasSubaccount(ctx, recipientID) {
		return true, types.ErrTypedTransferSubaccountNotFound
	}
	if senderID.Owner != recipientID.Owner {
		return true, types.ErrTypedTransferOwnerMismatch
	}
	var validateAsset func(sdk.Context, uint32) error
	switch {
	case senderType == types.AccountType_ACCOUNT_TYPE_FUNDING &&
		recipientType == types.AccountType_ACCOUNT_TYPE_SPOT:
		validateAsset = k.assetsKeeper.ValidateAssetForSpotTrading
	case senderType == types.AccountType_ACCOUNT_TYPE_SPOT &&
		recipientType == types.AccountType_ACCOUNT_TYPE_FUNDING:
		validateAsset = k.assetsKeeper.ValidateAssetForWithdrawal
	case senderType == types.AccountType_ACCOUNT_TYPE_FUNDING &&
		recipientType == types.AccountType_ACCOUNT_TYPE_PERPETUAL:
		validateAsset = k.assetsKeeper.ValidateAssetForPerpetual
	case senderType == types.AccountType_ACCOUNT_TYPE_PERPETUAL &&
		recipientType == types.AccountType_ACCOUNT_TYPE_FUNDING:
		validateAsset = k.assetsKeeper.ValidateAssetForWithdrawal
	default:
		return true, errorsmod.Wrapf(
			types.ErrTypedTransferRouteNotAllowed,
			"%s to %s",
			senderType.String(),
			recipientType.String(),
		)
	}

	if senderIsBusiness && !k.isCanonicalBusinessSubaccount(ctx, senderID, senderType) {
		return true, types.ErrTypedTransferMappingInvalid
	}
	if recipientIsBusiness && !k.isCanonicalBusinessSubaccount(ctx, recipientID, recipientType) {
		return true, types.ErrTypedTransferMappingInvalid
	}
	return true, validateAsset(ctx, assetID)
}

func (k Keeper) isCanonicalBusinessSubaccount(
	ctx sdk.Context,
	id types.SubaccountId,
	accountType types.AccountType,
) bool {
	mappedID, exists := k.GetTypedSubaccount(ctx, id.Owner, accountType)
	return exists && mappedID == id
}

func (k Keeper) getTransferAccountType(
	ctx sdk.Context,
	id types.SubaccountId,
	storedType types.AccountType,
) types.AccountType {
	if storedType != types.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return storedType
	}
	for _, accountType := range []types.AccountType{
		types.AccountType_ACCOUNT_TYPE_SPOT,
		types.AccountType_ACCOUNT_TYPE_FUNDING,
	} {
		mappedID, exists := k.GetTypedSubaccount(ctx, id.Owner, accountType)
		if exists && mappedID == id {
			return accountType
		}
	}
	return storedType
}
