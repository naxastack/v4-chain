package types_test

import (
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestValidateBusinessAccountType(t *testing.T) {
	tests := []struct {
		name        string
		accountType types.AccountType
		wantErr     bool
	}{
		{name: "spot", accountType: types.AccountType_ACCOUNT_TYPE_SPOT},
		{name: "funding", accountType: types.AccountType_ACCOUNT_TYPE_FUNDING},
		{name: "unspecified", accountType: types.AccountType_ACCOUNT_TYPE_UNSPECIFIED, wantErr: true},
		{name: "perpetual", accountType: types.AccountType_ACCOUNT_TYPE_PERPETUAL, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateBusinessAccountType(tt.accountType)
			if tt.wantErr {
				require.ErrorIs(t, err, types.ErrInvalidAccountType)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateAccountTypeTransition(t *testing.T) {
	tests := []struct {
		name    string
		current types.AccountType
		next    types.AccountType
		wantErr error
	}{
		{name: "finalize perpetual", current: types.AccountType_ACCOUNT_TYPE_UNSPECIFIED, next: types.AccountType_ACCOUNT_TYPE_PERPETUAL},
		{name: "finalize spot", current: types.AccountType_ACCOUNT_TYPE_UNSPECIFIED, next: types.AccountType_ACCOUNT_TYPE_SPOT},
		{name: "same perpetual", current: types.AccountType_ACCOUNT_TYPE_PERPETUAL, next: types.AccountType_ACCOUNT_TYPE_PERPETUAL},
		{name: "reject spot to funding", current: types.AccountType_ACCOUNT_TYPE_SPOT, next: types.AccountType_ACCOUNT_TYPE_FUNDING, wantErr: types.ErrAccountTypeImmutable},
		{name: "reject funding to perpetual", current: types.AccountType_ACCOUNT_TYPE_FUNDING, next: types.AccountType_ACCOUNT_TYPE_PERPETUAL, wantErr: types.ErrAccountTypeImmutable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateAccountTypeTransition(tt.current, tt.next)
			if tt.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
