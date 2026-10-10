package cli

import "testing"

func TestSpotQueryCommandsAreRegistered(t *testing.T) {
	requireCommandNames(t, GetQueryCmd("subaccounts"), "show-subaccount", "spot-order-epoch")
}
