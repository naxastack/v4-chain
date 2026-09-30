package keeper

import (
	"errors"
	"testing"

	errorsmod "cosmossdk.io/errors"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

func TestGetRecoverableSpotSkipReason(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantReason types.SpotOperationSkipReason
		wantOK     bool
	}{
		{
			name:       "insufficient balance",
			err:        satypes.ErrBusinessAssetPositionNegative,
			wantReason: types.SpotOperationSkipReasonInsufficientBalance,
			wantOK:     true,
		},
		{
			name:       "disabled asset through wrapping",
			err:        errorsmod.Wrap(assettypes.ErrAssetSpotTradingDisabled, "base asset"),
			wantReason: types.SpotOperationSkipReasonAssetNotTradable,
			wantOK:     true,
		},
		{
			name:       "fee cap",
			err:        types.ErrSpotFeeCapExceeded,
			wantReason: types.SpotOperationSkipReasonFeeCapExceeded,
			wantOK:     true,
		}, {
			name:       "stale short-term epoch",
			err:        types.ErrSpotOrderEpochMismatch,
			wantReason: types.SpotOperationSkipReasonOrderEpochMismatch,
			wantOK:     true,
		},
		{
			name:   "reservation invariant remains strict",
			err:    types.ErrInvalidOrderRemoval,
			wantOK: false,
		},
		{
			name:   "internal bank failure remains strict",
			err:    errors.New("bank failure"),
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reason, ok := getRecoverableSpotSkipReason(tc.err)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantReason, reason)
		})
	}
}
