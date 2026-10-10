package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clobcli "github.com/dydxprotocol/v4-chain/protocol/x/clob/client/cli"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

func MsgPlaceSpotOrderExec(
	clientCtx client.Context,
	fromName string,
	owner sdk.AccAddress,
	number uint32,
	clientID uint32,
	clobPairID uint32,
	side types.Order_Side,
	quantums uint64,
	subticks uint64,
	goodTil uint32,
	maxFeePpm uint32,
	orderEpoch uint64,
) (testutil.BufferWriter, error) {
	sideNum := 1
	if side == types.Order_SIDE_SELL {
		sideNum = 2
	}
	args := []string{
		owner.String(),
		fmt.Sprint(number),
		fmt.Sprint(clientID),
		fmt.Sprint(clobPairID),
		fmt.Sprint(sideNum),
		fmt.Sprint(quantums),
		fmt.Sprint(subticks),
		fmt.Sprint(goodTil),
		fmt.Sprint(maxFeePpm),
		fmt.Sprintf("--%s=%d", "order-epoch", orderEpoch),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, clobcli.CmdPlaceSpotOrder(), appendNode(clientCtx, args))
}

func MsgCancelSpotOrderExec(
	clientCtx client.Context,
	fromName string,
	owner sdk.AccAddress,
	number uint32,
	clientID uint32,
	clobPairID uint32,
	goodTil uint32,
) (testutil.BufferWriter, error) {
	args := []string{
		owner.String(),
		fmt.Sprint(number),
		fmt.Sprint(clientID),
		fmt.Sprint(clobPairID),
		fmt.Sprint(goodTil),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, clobcli.CmdCancelSpotOrder(), appendNode(clientCtx, args))
}

func MsgQueryOrderFillAmountExec(
	clientCtx client.Context,
	owner sdk.AccAddress,
	number uint32,
	clientID uint32,
	clobPairID uint32,
	orderFlags uint32,
) (testutil.BufferWriter, error) {
	args := []string{
		owner.String(),
		fmt.Sprint(number),
		fmt.Sprint(clientID),
		fmt.Sprint(clobPairID),
		fmt.Sprint(orderFlags),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, clobcli.CmdQueryOrderFillAmount(), appendNode(clientCtx, args))
}

func appendNode(clientCtx client.Context, args []string) []string {
	if clientCtx.NodeURI != "" {
		args = append(args, fmt.Sprintf("--%s=%s", flags.FlagNode, clientCtx.NodeURI))
	}
	return args
}
