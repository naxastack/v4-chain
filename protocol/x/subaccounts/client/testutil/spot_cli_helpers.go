package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sacli "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/client/cli"
)

func MsgCreateSubaccountExec(
	clientCtx client.Context,
	fromName string,
	accountType string,
) (testutil.BufferWriter, error) {
	args := []string{
		accountType,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, sacli.CmdCreateSubaccount(), appendNode(clientCtx, args))
}

func MsgQuerySpotSubaccountExec(
	clientCtx client.Context,
	owner sdk.AccAddress,
	number uint32,
) (testutil.BufferWriter, error) {
	args := []string{
		owner.String(),
		fmt.Sprint(number),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, sacli.CmdShowSubaccount(), appendNode(clientCtx, args))
}

func appendNode(clientCtx client.Context, args []string) []string {
	if clientCtx.NodeURI != "" {
		args = append(args, fmt.Sprintf("--%s=%s", flags.FlagNode, clientCtx.NodeURI))
	}
	return args
}
