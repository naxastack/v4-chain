package types

import "github.com/dydxprotocol/v4-chain/protocol/lib"

const (
	// UusdcDenom is the precomputed denom for IBC Micro USDC.
	UusdcDenom         = "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5"
	UusdcDenomExponent = -6
)

var (
	AssetUsdc = Asset{
		Id:               0,
		Symbol:           "USDC",
		DenomExponent:    UusdcDenomExponent,
		Denom:            UusdcDenom,
		HasMarket:        false,
		AtomicResolution: lib.QuoteCurrencyAtomicResolution,
	}
	AssetPolicyUsdc = AssetPolicy{
		AssetId:            AssetUsdc.Id,
		Status:             AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		DepositsEnabled:    true,
		WithdrawalsEnabled: true,
		SpotTradingEnabled: true,
		PerpetualEnabled:   true,
	}
)

// DefaultGenesis returns the default Capability genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Assets: []Asset{
			AssetUsdc,
		},
		AssetPolicies: []AssetPolicy{
			AssetPolicyUsdc,
		},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if len(gs.Assets) == 0 {
		return ErrNoAssetInGenesis
	}
	if gs.Assets[0] != AssetUsdc {
		return ErrUsdcMustBeAssetZero
	}

	assetIdSet := make(map[uint32]struct{})
	denomSet := make(map[string]struct{})
	expectedId := uint32(0)

	for _, asset := range gs.Assets {
		if _, exists := assetIdSet[asset.Id]; exists {
			return ErrAssetIdAlreadyExists
		}
		if _, exists := denomSet[asset.Denom]; exists {
			return ErrAssetDenomAlreadyExists
		}
		if asset.Id != expectedId {
			return ErrGapFoundInAssetId
		}
		if !asset.HasMarket && asset.MarketId > 0 {
			return ErrInvalidMarketId
		}
		assetIdSet[asset.Id] = struct{}{}
		denomSet[asset.Denom] = struct{}{}
		expectedId++
	}

	policyIds := make(map[uint32]struct{}, len(gs.AssetPolicies))
	for _, policy := range gs.AssetPolicies {
		if _, exists := assetIdSet[policy.AssetId]; !exists {
			return ErrAssetDoesNotExist
		}
		if _, exists := policyIds[policy.AssetId]; exists {
			return ErrAssetPolicyAlreadyExists
		}
		if policy.Status != AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE {
			return ErrInvalidAssetPolicyStatus
		}
		if err := ValidateAssetPolicy(policy); err != nil {
			return err
		}
		policyIds[policy.AssetId] = struct{}{}
	}
	if len(policyIds) != len(assetIdSet) {
		return ErrAssetPolicyDoesNotExist
	}
	return nil
}
