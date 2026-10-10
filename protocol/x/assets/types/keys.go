package types

// Module name and store keys
const (
	// ModuleName defines the module name
	ModuleName = "assets"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// State
const (
	// AssetKeyPrefix is the prefix to retrieve all Assets.
	AssetKeyPrefix = "Asset:"
	// AssetPolicyKeyPrefix is the prefix to retrieve all AssetPolicies.
	AssetPolicyKeyPrefix = "AssetPolicy:"
	// AssetPolicyStateKeyPrefix stores policy and lifecycle versions by asset id.
	AssetPolicyStateKeyPrefix = "AssetPolicyState:"
	// AssetLifecycleTaskKeyPrefix stores one lifecycle task per asset id.
	AssetLifecycleTaskKeyPrefix = "AssetLifecycleTask:"
	// AssetLifecycleSchedulerCursorKey stores the next asset id to service.
	AssetLifecycleSchedulerCursorKey = "AssetLifecycleSchedulerCursor"
)
