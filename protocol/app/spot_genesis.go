package app

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	clobtypes "github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// ValidateSpotGenesis checks cross-module Spot invariants after all modules
// have initialized. It is intentionally read-only so invalid genesis state
// fails startup instead of being repaired implicitly.
func (app *App) ValidateSpotGenesis(ctx sdk.Context) error {
	if err := app.ClobKeeper.GetSpotResourceParams(ctx).Validate(); err != nil {
		return fmt.Errorf("invalid spot resource params: %w", err)
	}
	if err := app.FeeTiersKeeper.GetSpotFeeParams(ctx).Validate(); err != nil {
		return fmt.Errorf("invalid spot fee params: %w", err)
	}
	if app.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName) == nil {
		return fmt.Errorf("fee collector module account is not initialized")
	}
	if app.AccountKeeper.GetModuleAccount(ctx, satypes.ModuleName) == nil {
		return fmt.Errorf("default collateral pool module account is not initialized")
	}

	hasSpotPair := false
	for _, pair := range app.ClobKeeper.GetAllClobPairs(ctx) {
		if pair.GetSpotClobMetadata() == nil {
			continue
		}
		hasSpotPair = true
		lifecycle, found := app.ClobKeeper.GetSpotPairLifecycleState(ctx, pair.GetClobPairId())
		if !found {
			return fmt.Errorf("spot CLOB pair %d has no lifecycle state", pair.Id)
		}
		if pair.Status == clobtypes.ClobPair_STATUS_ACTIVE {
			if lifecycle.PauseSource != clobtypes.SpotPauseSourceNone || lifecycle.FinalSettlementStarted {
				return fmt.Errorf("active spot CLOB pair %d has inconsistent lifecycle state", pair.Id)
			}
			if err := app.ClobKeeper.ValidateSpotMarketActivation(ctx, pair); err != nil {
				return fmt.Errorf("spot CLOB pair %d cannot be active: %w", pair.Id, err)
			}
		}
		if pair.Status == clobtypes.ClobPair_STATUS_FINAL_SETTLEMENT && !lifecycle.FinalSettlementStarted {
			return fmt.Errorf("final-settlement spot CLOB pair %d has no final-settlement marker", pair.Id)
		}
	}

	reservationTotals := make(map[string]*big.Int)
	for _, reservation := range app.ClobKeeper.GetAllStatefulSpotReservations(ctx) {
		pair, found := app.ClobKeeper.GetClobPair(ctx, clobtypes.ClobPairId(reservation.OrderId.ClobPairId))
		if !found || pair.GetSpotClobMetadata() == nil {
			return fmt.Errorf("spot reservation references invalid CLOB pair %d", reservation.OrderId.ClobPairId)
		}
		account := app.SubaccountsKeeper.GetSubaccount(ctx, reservation.OrderId.SubaccountId)
		if account.AccountType != satypes.AccountType_ACCOUNT_TYPE_SPOT {
			return fmt.Errorf("spot reservation references non-spot subaccount %s", reservation.OrderId.SubaccountId.String())
		}
		if !app.ClobKeeper.HasStatefulSpotOrderMarketIndex(ctx, reservation.OrderId) {
			return fmt.Errorf("spot reservation %s has no market index", reservation.OrderId.String())
		}
		metadata := pair.GetSpotClobMetadata()
		if reservation.OutgoingAssetId != metadata.BaseAssetId && reservation.OutgoingAssetId != metadata.QuoteAssetId {
			return fmt.Errorf("spot reservation %s has invalid outgoing asset %d", reservation.OrderId.String(), reservation.OutgoingAssetId)
		}
		key := spotReservationAggregateKey(reservation.OrderId.SubaccountId, reservation.OutgoingAssetId)
		if _, exists := reservationTotals[key]; !exists {
			reservationTotals[key] = new(big.Int)
		}
		reservationTotals[key].Add(reservationTotals[key], reservation.ReservedQuantums.BigInt())
	}

	for _, account := range app.SubaccountsKeeper.GetAllSubaccount(ctx) {
		if account.Id == nil {
			return fmt.Errorf("subaccount has no id")
		}
		if account.AccountType.IsBusinessAccountType() {
			mapped, found := app.SubaccountsKeeper.GetTypedSubaccount(ctx, account.Id.Owner, account.AccountType)
			if !found || mapped != *account.Id {
				return fmt.Errorf("subaccount %s has inconsistent typed account mapping", account.Id.String())
			}
		}
		for _, position := range account.AssetPositions {
			key := spotReservationAggregateKey(*account.Id, position.AssetId)
			expected := reservationTotals[key]
			if expected == nil {
				expected = new(big.Int)
			}
			if position.StatefulReservedQuantums.IsNil() ||
				position.StatefulReservedQuantums.BigInt().Cmp(expected) != 0 {
				return fmt.Errorf("subaccount %s asset %d reservation aggregate does not match reservation details", account.Id.String(), position.AssetId)
			}
			delete(reservationTotals, key)
		}
	}
	if len(reservationTotals) != 0 {
		return fmt.Errorf("reservation aggregate references a missing asset position")
	}

	for _, mapping := range app.SubaccountsKeeper.GetAllTypedSubaccounts(ctx) {
		if mapping.SubaccountId == nil {
			return fmt.Errorf("typed account mapping for %s has no subaccount id", mapping.Owner)
		}
		account := app.SubaccountsKeeper.GetSubaccount(ctx, *mapping.SubaccountId)
		if account.Id == nil || account.AccountType != mapping.AccountType || account.Id.Owner != mapping.Owner {
			return fmt.Errorf("typed account mapping for %s references an inconsistent subaccount", mapping.Owner)
		}
	}

	for _, epoch := range app.SubaccountsKeeper.GetAllSpotOrderEpochs(ctx) {
		if epoch.SubaccountId == nil {
			return fmt.Errorf("spot order epoch has no subaccount id")
		}
		account := app.SubaccountsKeeper.GetSubaccount(ctx, *epoch.SubaccountId)
		if account.Id == nil || account.AccountType != satypes.AccountType_ACCOUNT_TYPE_SPOT {
			return fmt.Errorf("spot order epoch references non-spot subaccount %s", epoch.SubaccountId.String())
		}
		if _, found := app.AssetsKeeper.GetAsset(ctx, epoch.AssetId); !found {
			return fmt.Errorf("spot order epoch references unknown asset %d", epoch.AssetId)
		}
	}

	if !hasSpotPair {
		return nil
	}

	poolAddresses := map[string]struct{}{satypes.ModuleAddress.String(): {}}
	for _, perpetual := range app.PerpetualsKeeper.GetAllPerpetuals(ctx) {
		poolAddress, err := app.SubaccountsKeeper.GetCollateralPoolFromPerpetualId(ctx, perpetual.Params.Id)
		if err != nil {
			return fmt.Errorf("cannot resolve collateral pool for perpetual %d: %w", perpetual.Params.Id, err)
		}
		poolAddresses[poolAddress.String()] = struct{}{}
	}

	for _, asset := range app.AssetsKeeper.GetAllAssets(ctx) {
		poolQuantums, err := app.SubaccountsKeeper.GetCollateralPoolAssetQuantums(ctx, asset.Id)
		if err != nil {
			return fmt.Errorf("cannot calculate collateral pool balance for asset %d: %w", asset.Id, err)
		}
		for pool := range poolAddresses {
			if _, found := poolQuantums[pool]; !found {
				poolQuantums[pool] = new(big.Int)
			}
		}
		for pool, quantums := range poolQuantums {
			if quantums.Sign() < 0 {
				return fmt.Errorf("collateral pool %s has negative ledger balance for asset %d", pool, asset.Id)
			}
			converted, coin, err := app.AssetsKeeper.ConvertAssetToCoin(ctx, asset.Id, quantums)
			if err != nil {
				return fmt.Errorf("cannot convert collateral pool balance for asset %d: %w", asset.Id, err)
			}
			if converted.Cmp(quantums) != 0 {
				return fmt.Errorf("collateral pool %s balance for asset %d is not exactly representable", pool, asset.Id)
			}
			poolAddress, err := sdk.AccAddressFromBech32(pool)
			if err != nil {
				return fmt.Errorf("invalid collateral pool address %s: %w", pool, err)
			}
			bankBalance := app.BankKeeper.GetBalance(ctx, poolAddress, asset.Denom)
			if !bankBalance.Amount.Equal(coin.Amount) {
				return fmt.Errorf("collateral pool balance mismatch for pool %s denom %s: ledger=%s bank=%s", pool, asset.Denom, coin.Amount, bankBalance.Amount)
			}
		}
	}

	return nil
}

func spotReservationAggregateKey(id satypes.SubaccountId, assetId uint32) string {
	return fmt.Sprintf("%s/%d", id.String(), assetId)
}
