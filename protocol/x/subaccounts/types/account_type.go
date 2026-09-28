package types

// IsBusinessAccountType reports whether the account type can be explicitly created.
func (accountType AccountType) IsBusinessAccountType() bool {
	return accountType == AccountType_ACCOUNT_TYPE_SPOT || accountType == AccountType_ACCOUNT_TYPE_FUNDING
}

// ValidateBusinessAccountType validates an account type for explicit creation.
func ValidateBusinessAccountType(accountType AccountType) error {
	if !accountType.IsBusinessAccountType() {
		return ErrInvalidAccountType
	}
	return nil
}

// ValidateAccountTypeTransition validates a persisted account type transition.
// UNSPECIFIED is reserved for an untyped in-memory account and may be finalized once.
func ValidateAccountTypeTransition(current AccountType, next AccountType) error {
	if next != AccountType_ACCOUNT_TYPE_UNSPECIFIED &&
		next != AccountType_ACCOUNT_TYPE_PERPETUAL &&
		next != AccountType_ACCOUNT_TYPE_SPOT &&
		next != AccountType_ACCOUNT_TYPE_FUNDING {
		return ErrInvalidAccountType
	}
	if current != AccountType_ACCOUNT_TYPE_UNSPECIFIED && current != next {
		return ErrAccountTypeImmutable
	}
	return nil
}
