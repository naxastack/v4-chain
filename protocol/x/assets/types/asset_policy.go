package types

// ValidateAssetPolicy validates a policy independently of module state.
func ValidateAssetPolicy(policy AssetPolicy) error {
	switch policy.Status {
	case AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
		AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED:
	default:
		return ErrInvalidAssetPolicyStatus
	}
	if !policy.WithdrawalsEnabled {
		return ErrAssetExitMustRemainEnabled
	}
	return nil
}

// ValidateAssetPolicyTransition validates a state transition for one asset.
func ValidateAssetPolicyTransition(previous, next AssetPolicy) error {
	if previous.AssetId != next.AssetId {
		return ErrAssetPolicyAssetIdMismatch
	}
	if err := ValidateAssetPolicy(next); err != nil {
		return err
	}
	if previous.Status == AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED &&
		next.Status != AssetPolicyStatus_ASSET_POLICY_STATUS_RETIRED {
		return ErrInvalidAssetPolicyTransition
	}
	return nil
}

// AllowsDeposit reports whether a new bank-to-funding deposit is permitted.
func (policy AssetPolicy) AllowsDeposit() bool {
	return policy.Status == AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE &&
		policy.DepositsEnabled
}

// AllowsWithdrawal reports whether an exit to a bank address is permitted.
func (policy AssetPolicy) AllowsWithdrawal() bool {
	return policy.Status != AssetPolicyStatus_ASSET_POLICY_STATUS_UNSPECIFIED &&
		policy.WithdrawalsEnabled
}

// AllowsSpotTrading reports whether spot transfers, orders, and fills are permitted.
func (policy AssetPolicy) AllowsSpotTrading() bool {
	return policy.Status == AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE &&
		policy.SpotTradingEnabled
}

// AllowsPerpetual reports whether existing perpetual paths may use the asset.
func (policy AssetPolicy) AllowsPerpetual() bool {
	return policy.Status == AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE &&
		policy.PerpetualEnabled
}
