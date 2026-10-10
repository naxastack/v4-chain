package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpotQueryCommandsAreRegistered(t *testing.T) {
	cmd := GetQueryCmd("clob")
	for _, name := range []string{
		"spot-resource-params",
		"spot-stateful-orders",
		"stateful-spot-reservation",
		"spot-pair-lifecycle",
		"spot-delist-cursor",
		"order-fill-amount",
		"show-clob-pair",
		"stateful-order",
	} {
		require.NotNil(t, cmd.Commands(), name)
		found := false
		for _, child := range cmd.Commands() {
			if child.Name() == name {
				found = true
			}
		}
		require.True(t, found, name)
	}
}
