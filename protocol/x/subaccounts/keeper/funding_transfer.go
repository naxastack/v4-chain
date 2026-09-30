package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// DepositFundsToFundingAccount atomically moves bank funds into an existing
// canonical FUNDING subaccount owned by the bank sender.
func (k Keeper) DepositFundsToFundingAccount(
	ctx sdk.Context,
	fromAccount sdk.AccAddress,
	toSubaccountID types.SubaccountId,
	assetID uint32,
	quantums *big.Int,
) error {
	cacheCtx, write := ctx.CacheContext()

	if fromAccount.String() != toSubaccountID.Owner {
		return types.ErrFundingDepositOwnerMismatch
	}
	if err := k.validateFundingSubaccount(cacheCtx, toSubaccountID); err != nil {
		return err
	}
	if err := k.assetsKeeper.ValidateAssetForDeposit(cacheCtx, assetID); err != nil {
		return err
	}
	if err := k.depositFundsFromAccountToSubaccount(
		cacheCtx,
		fromAccount,
		toSubaccountID,
		assetID,
		quantums,
		true,
	); err != nil {
		return err
	}

	write()
	return nil
}

// WithdrawFundsFromFundingAccount atomically moves funds from an existing
// canonical FUNDING subaccount to a bank account.
func (k Keeper) WithdrawFundsFromFundingAccount(
	ctx sdk.Context,
	fromSubaccountID types.SubaccountId,
	toAccount sdk.AccAddress,
	assetID uint32,
	quantums *big.Int,
) error {
	cacheCtx, write := ctx.CacheContext()

	if err := k.validateFundingSubaccount(cacheCtx, fromSubaccountID); err != nil {
		return err
	}
	if err := k.assetsKeeper.ValidateAssetForWithdrawal(cacheCtx, assetID); err != nil {
		return err
	}
	if err := k.withdrawFundsFromSubaccountToAccount(
		cacheCtx,
		fromSubaccountID,
		toAccount,
		assetID,
		quantums,
		true,
	); err != nil {
		return err
	}

	write()
	return nil
}

func (k Keeper) validateFundingSubaccount(ctx sdk.Context, id types.SubaccountId) error {
	if !k.hasSubaccount(ctx, id) {
		return types.ErrFundingSubaccountNotFound
	}
	subaccount := k.GetSubaccount(ctx, id)
	if subaccount.AccountType != types.AccountType_ACCOUNT_TYPE_FUNDING {
		return types.ErrFundingSubaccountRequired
	}
	if !k.isCanonicalBusinessSubaccount(
		ctx,
		id,
		types.AccountType_ACCOUNT_TYPE_FUNDING,
	) {
		return types.ErrTypedTransferMappingInvalid
	}
	return nil
}
