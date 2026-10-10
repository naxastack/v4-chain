package testutil

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sendingcli "github.com/dydxprotocol/v4-chain/protocol/x/sending/client/cli"
)

func MsgDepositAssetExec(
	clientCtx client.Context,
	sender sdk.AccAddress,
	recipient sdk.AccAddress,
	recipientNumber uint32,
	quantums uint64,
	assetID uint32,
) (testutil.BufferWriter, error) {
	args := []string{
		sender.String(),
		recipient.String(),
		fmt.Sprint(recipientNumber),
		fmt.Sprint(quantums),
		fmt.Sprintf("--%s=%d", "asset-id", assetID),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, sendingcli.CmdDepositToSubaccount(), appendNode(clientCtx, args))
}

func MsgCreateTransferAssetExec(
	clientCtx client.Context,
	fromName string,
	sender sdk.AccAddress,
	senderNumber uint32,
	recipient sdk.AccAddress,
	recipientNumber uint32,
	amount uint64,
	assetID uint32,
) (testutil.BufferWriter, error) {
	args := []string{
		sender.String(),
		fmt.Sprint(senderNumber),
		recipient.String(),
		fmt.Sprint(recipientNumber),
		fmt.Sprint(amount),
		fmt.Sprintf("--%s=%d", "asset-id", assetID),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, fromName),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
	}
	return clitestutil.ExecTestCLICmd(clientCtx, sendingcli.CmdCreateTransfer(), appendNode(clientCtx, args))
}

func appendNode(clientCtx client.Context, args []string) []string {
	if clientCtx.NodeURI != "" {
		args = append(args, fmt.Sprintf("--%s=%s", flags.FlagNode, clientCtx.NodeURI))
	}
	return args
}
