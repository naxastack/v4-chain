package cli

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	flagSpotLongTerm   = "long-term"
	flagSpotOrderEpoch = "order-epoch"
)

type sdkMsg interface {
	sdk.Msg
}
