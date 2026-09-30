package keeper

import (
	"fmt"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// processSpotSingleMatch atomically applies the consensus state changes for one
// spot match. Final external fill events remain the responsibility of the
// proposer-operation result path.
func (k Keeper) processSpotSingleMatch(
	ctx sdk.Context,
	matchWithOrders *types.MatchWithOrders,
	clobPair types.ClobPair,
	settlement types.SpotMatchSettlement,
) (
	bool,
	satypes.UpdateResult,
	satypes.UpdateResult,
	*big.Int,
	error,
) {
	var takerResult, makerResult satypes.UpdateResult
	zeroAffiliateShare := new(big.Int)

	makerOrder := matchWithOrders.MakerOrder.MustGetOrder()
	takerOrder := matchWithOrders.TakerOrder.MustGetOrder()
	if err := k.validateSpotShortTermOrderEpoch(ctx, makerOrder, clobPair); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	if err := k.validateSpotShortTermOrderEpoch(ctx, takerOrder, clobPair); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	_, currentMakerFill, makerPruneableBlockHeight := k.GetOrderFillAmount(ctx, makerOrder.OrderId)
	_, currentTakerFill, takerPruneableBlockHeight := k.GetOrderFillAmount(ctx, takerOrder.OrderId)

	newMakerFill, err := getUpdatedOrderFillAmount(
		makerOrder.OrderId,
		matchWithOrders.MakerOrder.GetBaseQuantums(),
		currentMakerFill,
		matchWithOrders.FillAmount,
	)
	if err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	newTakerFill, err := getUpdatedOrderFillAmount(
		takerOrder.OrderId,
		matchWithOrders.TakerOrder.GetBaseQuantums(),
		currentTakerFill,
		matchWithOrders.FillAmount,
	)
	if err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}

	if err := k.assetsKeeper.ValidateAssetForSpotTrading(ctx, settlement.BaseAssetId); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	if err := k.assetsKeeper.ValidateAssetForSpotTrading(ctx, settlement.QuoteAssetId); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	makerSubaccount := k.subaccountsKeeper.GetSubaccount(ctx, makerOrder.OrderId.SubaccountId)
	takerSubaccount := k.subaccountsKeeper.GetSubaccount(ctx, takerOrder.OrderId.SubaccountId)
	if makerSubaccount.AccountType != satypes.AccountType_ACCOUNT_TYPE_SPOT ||
		takerSubaccount.AccountType != satypes.AccountType_ACCOUNT_TYPE_SPOT {
		return false, takerResult, makerResult, zeroAffiliateShare,
			errorsmod.Wrap(satypes.ErrInvalidAccountType, "spot matches require spot subaccounts")
	}

	cacheCtx, write := ctx.CacheContext()
	if makerOrder.IsStatefulOrder() {
		if err := k.consumeStatefulSpotReservationForMatch(cacheCtx, makerOrder, clobPair, newMakerFill); err != nil {
			return false, takerResult, makerResult, zeroAffiliateShare, err
		}
	}
	if takerOrder.IsStatefulOrder() {
		if err := k.consumeStatefulSpotReservationForMatch(cacheCtx, takerOrder, clobPair, newTakerFill); err != nil {
			return false, takerResult, makerResult, zeroAffiliateShare, err
		}
	}

	buyerId := takerOrder.OrderId.SubaccountId
	sellerId := makerOrder.OrderId.SubaccountId
	if settlement.MakerIsBuyer {
		buyerId, sellerId = sellerId, buyerId
	}
	updates := []satypes.Update{
		{
			SubaccountId: buyerId,
			AssetUpdates: []satypes.AssetUpdate{
				{AssetId: settlement.BaseAssetId, BigQuantumsDelta: new(big.Int).Set(settlement.BuyerBaseDelta)},
				{AssetId: settlement.QuoteAssetId, BigQuantumsDelta: new(big.Int).Set(settlement.BuyerQuoteDelta)},
			},
		},
		{
			SubaccountId: sellerId,
			AssetUpdates: []satypes.AssetUpdate{
				{AssetId: settlement.BaseAssetId, BigQuantumsDelta: new(big.Int).Set(settlement.SellerBaseDelta)},
				{AssetId: settlement.QuoteAssetId, BigQuantumsDelta: new(big.Int).Set(settlement.SellerQuoteDelta)},
			},
		},
	}
	if !settlement.BuyerFeeQuoteQuantums.IsInt64() || !settlement.SellerFeeQuoteQuantums.IsInt64() {
		return false, takerResult, makerResult, zeroAffiliateShare,
			errorsmod.Wrap(types.ErrInvalidPlaceOrder, "spot fee does not fit the match fee fields")
	}
	totalFees := new(big.Int).Add(
		new(big.Int).Set(settlement.BuyerFeeQuoteQuantums),
		settlement.SellerFeeQuoteQuantums,
	)
	if err := k.subaccountsKeeper.TransferSpotFees(
		cacheCtx,
		settlement.QuoteAssetId,
		totalFees,
	); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}

	// UpdateSubaccounts emits the final subaccount update events. Keep it after
	// all fallible reservation and bank operations so no event can escape a
	// later failure in this cached transaction.
	success, updateResults, err := k.subaccountsKeeper.UpdateSubaccounts(cacheCtx, updates, satypes.SpotMatch)
	if err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	if err := satypes.GetErrorFromUpdateResults(success, updateResults, updates); err != nil {
		return false, takerResult, makerResult, zeroAffiliateShare, err
	}
	if len(updateResults) != 2 {
		return false, takerResult, makerResult, zeroAffiliateShare,
			fmt.Errorf("spot match returned %d update results, expected 2", len(updateResults))
	}
	if settlement.MakerIsBuyer {
		makerResult, takerResult = updateResults[0], updateResults[1]
	} else {
		takerResult, makerResult = updateResults[0], updateResults[1]
	}

	matchWithOrders.MakerFee = settlement.SellerFeeQuoteQuantums.Int64()
	matchWithOrders.TakerFee = settlement.BuyerFeeQuoteQuantums.Int64()
	if settlement.MakerIsBuyer {
		matchWithOrders.MakerFee, matchWithOrders.TakerFee =
			settlement.BuyerFeeQuoteQuantums.Int64(), settlement.SellerFeeQuoteQuantums.Int64()
	}

	k.setOrderFillAmountsAndPruning(cacheCtx, makerOrder, newMakerFill, makerPruneableBlockHeight)
	k.setOrderFillAmountsAndPruning(cacheCtx, takerOrder, newTakerFill, takerPruneableBlockHeight)
	write()
	return true, takerResult, makerResult, zeroAffiliateShare, nil
}

func (k Keeper) consumeStatefulSpotReservationForMatch(
	ctx sdk.Context,
	order types.Order,
	pair types.ClobPair,
	newTotalFill satypes.BaseQuantums,
) error {
	reservation, found := k.GetStatefulSpotReservation(ctx, order.OrderId)
	if !found {
		return types.ErrStatefulOrderDoesNotExist
	}
	if newTotalFill.ToUint64() > order.Quantums {
		return errorsmod.Wrap(types.ErrInvalidPlaceOrder, "spot fill exceeds order quantums")
	}
	remaining := order.Quantums - newTotalFill.ToUint64()
	assetId, required, err := types.CalculateSpotOrderReservationForRemaining(order, pair, remaining)
	if err != nil {
		return err
	}
	if assetId != reservation.OutgoingAssetId {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation asset changed during settlement")
	}
	current := reservation.ReservedQuantums.BigInt()
	if current == nil || current.Sign() <= 0 || required.Cmp(current) > 0 {
		return errorsmod.Wrap(types.ErrInvalidOrderRemoval, "spot reservation cannot increase during settlement")
	}
	release := new(big.Int).Sub(new(big.Int).Set(current), required)
	if release.Sign() > 0 {
		if err := k.subaccountsKeeper.ReleaseStatefulSpotQuantums(
			ctx,
			order.OrderId.SubaccountId,
			assetId,
			release,
		); err != nil {
			return err
		}
	}
	reservation.ReservedQuantums = dtypes.NewIntFromBigInt(required)
	reservation.RemainingBaseQuantums = remaining
	k.SetStatefulSpotReservation(ctx, reservation)
	return nil
}

func (k Keeper) validateSpotShortTermOrderEpoch(
	ctx sdk.Context,
	order types.Order,
	pair types.ClobPair,
) error {
	if !order.IsShortTermOrder() {
		return nil
	}
	metadata := pair.GetSpotClobMetadata()
	if metadata == nil {
		return errorsmod.Wrap(types.ErrInvalidClobPairParameter, "spot epoch validation requires a spot CLOB pair")
	}
	assetId := metadata.QuoteAssetId
	if !order.IsBuy() {
		assetId = metadata.BaseAssetId
	}
	currentEpoch := k.subaccountsKeeper.GetSpotOrderEpoch(ctx, order.OrderId.SubaccountId, assetId)
	if order.OrderEpoch != currentEpoch {
		return errorsmod.Wrapf(
			types.ErrSpotOrderEpochMismatch,
			"order epoch %d does not match current epoch %d for asset %d",
			order.OrderEpoch,
			currentEpoch,
			assetId,
		)
	}
	return nil
}
