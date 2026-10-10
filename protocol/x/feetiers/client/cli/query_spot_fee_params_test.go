package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpotFeeParamsCommandIsRegistered(t *testing.T) {
	cmd := GetQueryCmd("feetiers")
	found := false
	for _, child := range cmd.Commands() {
		if child.Name() == "get-spot-fee-params" {
			found = true
		}
	}
	require.True(t, found)
}
