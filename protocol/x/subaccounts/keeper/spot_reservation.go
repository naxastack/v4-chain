package keeper

import (
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// ReserveStatefulSpotQuantums increases the locked portion of an existing asset position.
func (k Keeper) ReserveStatefulSpotQuantums(
	ctx sdk.Context,
	id types.SubaccountId,
	assetId uint32,
	amount *big.Int,
) error {
	if amount == nil || amount.Sign() <= 0 {
		return errorsmod.Wrap(types.ErrStatefulReservedQuantumsInvalid, "reservation must be positive")
	}
	subaccount := k.GetSubaccount(ctx, id)
	if subaccount.AccountType != types.AccountType_ACCOUNT_TYPE_SPOT {
		return errorsmod.Wrap(types.ErrInvalidAccountType, "spot reservation requires a spot subaccount")
	}

	for _, position := range subaccount.AssetPositions {
		if position.AssetId != assetId {
			continue
		}
		total := position.Quantums.BigInt()
		reserved := position.StatefulReservedQuantums.BigInt()
		if reserved == nil {
			reserved = new(big.Int)
		}
		next := new(big.Int).Add(new(big.Int).Set(reserved), amount)
		if total == nil || total.Sign() < 0 || next.Cmp(total) > 0 {
			return errorsmod.Wrap(types.ErrStatefulReservedQuantumsInvalid, "reservation exceeds available asset quantums")
		}
		position.StatefulReservedQuantums = dtypes.NewIntFromBigInt(next)
		if _, err := k.IncrementSpotOrderEpoch(ctx, id, assetId); err != nil {
			return err
		}
		k.SetSubaccount(ctx, subaccount)
		return nil
	}
	return errorsmod.Wrap(types.ErrAssetPositionNotSupported, "reservation asset position does not exist")
}

// ReleaseStatefulSpotQuantums decreases the locked portion without changing the asset epoch.
func (k Keeper) ReleaseStatefulSpotQuantums(
	ctx sdk.Context,
	id types.SubaccountId,
	assetId uint32,
	amount *big.Int,
) error {
	if amount == nil || amount.Sign() <= 0 {
		return errorsmod.Wrap(types.ErrStatefulReservedQuantumsInvalid, "release must be positive")
	}
	subaccount := k.GetSubaccount(ctx, id)
	if subaccount.AccountType != types.AccountType_ACCOUNT_TYPE_SPOT {
		return errorsmod.Wrap(types.ErrInvalidAccountType, "spot release requires a spot subaccount")
	}
	for _, position := range subaccount.AssetPositions {
		if position.AssetId != assetId {
			continue
		}
		reserved := position.StatefulReservedQuantums.BigInt()
		if reserved == nil || reserved.Cmp(amount) < 0 {
			return errorsmod.Wrap(types.ErrStatefulReservedQuantumsInvalid, "release exceeds reserved asset quantums")
		}
		next := new(big.Int).Sub(new(big.Int).Set(reserved), amount)
		position.StatefulReservedQuantums = dtypes.NewIntFromBigInt(next)
		k.SetSubaccount(ctx, subaccount)
		return nil
	}
	return errorsmod.Wrap(types.ErrAssetPositionNotSupported, "release asset position does not exist")
}
