package keeper_test

import (
	"testing"

	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestSetSubaccountLifecycleByAccountType(t *testing.T) {
	tests := []struct {
		name        string
		accountType types.AccountType
		wantExists  bool
	}{
		{name: "perpetual empty accounts are deleted", accountType: types.AccountType_ACCOUNT_TYPE_PERPETUAL, wantExists: false},
		{name: "spot empty accounts persist", accountType: types.AccountType_ACCOUNT_TYPE_SPOT, wantExists: true},
		{name: "funding empty accounts persist", accountType: types.AccountType_ACCOUNT_TYPE_FUNDING, wantExists: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, k, _, _, _, _, _, _, _, _, _ := keepertest.SubaccountsKeepers(t, true)
			id := types.SubaccountId{Owner: "dydx1lifecycleowner000000000000000000000000", Number: 1}
			k.SetSubaccount(ctx, types.Subaccount{Id: &id, AccountType: tt.accountType})
			stored := k.GetSubaccount(ctx, id)
			require.Equal(t, tt.wantExists, stored.AccountType == tt.accountType)
		})
	}
}
