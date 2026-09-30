package types

import (
	errorsmod "cosmossdk.io/errors"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
)

// DefaultGenesis returns the default Capability genesis state.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Subaccounts: []Subaccount{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	includedAccounts := make(map[SubaccountId]bool)
	accountsByID := make(map[SubaccountId]Subaccount)
	for _, sa := range gs.Subaccounts {
		subaccountId := sa.GetId()
		if subaccountId == nil {
			return ErrInvalidSubaccountIdOwner
		}
		if err := subaccountId.Validate(); err != nil {
			return err
		}
		if includedAccounts[*subaccountId] {
			return errorsmod.Wrapf(ErrDuplicateSubaccountIds,
				"duplicate subaccount id %+v found within genesis state", subaccountId)
		}
		includedAccounts[*subaccountId] = true
		accountsByID[*subaccountId] = sa

		// Validate AssetPositions.
		// TODO(DEC-582): once we support different assets, remove this validation.
		if len(sa.GetAssetPositions()) > 1 {
			return ErrMultAssetPositionsNotSupported
		}
		for i := 0; i < len(sa.GetAssetPositions()); i++ {
			assetP := sa.GetAssetPositions()[i]
			if assetP == nil {
				return ErrAssetPositionNotSupported
			}
			if i > 0 && assetP.AssetId <= sa.GetAssetPositions()[i-1].AssetId {
				return ErrAssetPositionsOutOfOrder
			}
			if assetP.AssetId != 0 {
				return ErrAssetPositionNotSupported
			}
			if assetP.GetBigQuantums().Sign() == 0 {
				return ErrAssetPositionZeroQuantum
			}
			reserved := assetP.StatefulReservedQuantums.BigInt()
			if reserved != nil && reserved.Sign() < 0 {
				return ErrStatefulReservedQuantumsInvalid
			}
			if reserved != nil && assetP.GetBigQuantums().Cmp(reserved) < 0 {
				return ErrStatefulReservedQuantumsInvalid
			}
		}

		// Validate PerpetualPositions.
		for i := 0; i < len(sa.GetPerpetualPositions()); i++ {
			perpP := sa.GetPerpetualPositions()[i]
			if perpP == nil {
				return ErrPerpPositionZeroQuantum
			}
			if i > 0 && perpP.PerpetualId <= sa.GetPerpetualPositions()[i-1].PerpetualId {
				return ErrPerpPositionsOutOfOrder
			}
			if perpP.GetBigQuantums().Sign() == 0 {
				return ErrPerpPositionZeroQuantum
			}
		}
	}

	typedKeys := make(map[string]bool)
	mappedSpotAccounts := make(map[SubaccountId]bool)
	for _, mapping := range gs.TypedSubaccounts {
		if !mapping.AccountType.IsBusinessAccountType() || mapping.SubaccountId == nil {
			return ErrGenesisTypedAccountInvalid
		}
		if err := mapping.SubaccountId.Validate(); err != nil {
			return errorsmod.Wrapf(ErrGenesisTypedAccountInvalid, "invalid mapped subaccount: %v", err)
		}
		key := mapping.Owner + ":" + mapping.AccountType.String()
		if typedKeys[key] {
			return ErrGenesisTypedAccountInvalid
		}
		typedKeys[key] = true
		sa, found := accountsByID[*mapping.SubaccountId]
		if !found || sa.GetId().Owner != mapping.Owner || sa.AccountType != mapping.AccountType {
			return ErrGenesisTypedAccountInvalid
		}
		if mapping.AccountType == AccountType_ACCOUNT_TYPE_SPOT {
			mappedSpotAccounts[*mapping.SubaccountId] = true
		}
	}

	cursorOwners := make(map[string]bool)
	maxBusinessNumber := make(map[string]uint32)
	for _, sa := range gs.Subaccounts {
		if sa.AccountType.IsBusinessAccountType() && sa.Id != nil && sa.Id.Number > maxBusinessNumber[sa.Id.Owner] {
			maxBusinessNumber[sa.Id.Owner] = sa.Id.Number
		}
	}
	for _, cursor := range gs.BusinessNumberCursors {
		if cursor.Owner == "" || cursor.NextNumber < 1 || cursor.NextNumber > MaxSubaccountIdNumber+1 ||
			cursor.NextNumber <= maxBusinessNumber[cursor.Owner] {
			return ErrGenesisBusinessCursorInvalid
		}
		if cursorOwners[cursor.Owner] {
			return ErrGenesisBusinessCursorInvalid
		}
		cursorOwners[cursor.Owner] = true
	}

	epochKeys := make(map[string]bool)
	for _, epoch := range gs.SpotOrderEpochs {
		if epoch.SubaccountId == nil || epoch.Epoch == 0 || !mappedSpotAccounts[*epoch.SubaccountId] {
			return ErrGenesisSpotOrderEpochInvalid
		}
		key := epoch.SubaccountId.String() + ":" + lib.UintToString(epoch.AssetId)
		if epochKeys[key] {
			return ErrGenesisSpotOrderEpochInvalid
		}
		epochKeys[key] = true
	}
	return nil
}
