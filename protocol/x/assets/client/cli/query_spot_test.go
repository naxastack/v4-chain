package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpotAssetQueryCommandsAreRegistered(t *testing.T) {
	cmd := GetQueryCmd("assets")
	for _, name := range []string{
		"asset-policy",
		"asset-policy-state",
		"asset-lifecycle-task",
		"asset-lifecycle-scheduler-cursor",
	} {
		found := false
		for _, child := range cmd.Commands() {
			if child.Name() == name {
				found = true
			}
		}
		require.True(t, found, name)
	}
}
