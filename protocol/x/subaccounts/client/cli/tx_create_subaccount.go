package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/spf13/cobra"
)

func CmdCreateSubaccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-subaccount spot|funding",
		Short: "Broadcast MsgCreateSubaccount. The owner is the --from address. The chain allocates the subaccount number.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			accountType, err := parseBusinessAccountType(args[0])
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			owner := clientCtx.GetFromAddress()
			if owner.Empty() {
				return fmt.Errorf("create-subaccount requires --from")
			}
			msg := &types.MsgCreateSubaccount{
				Owner:       owner.String(),
				AccountType: accountType,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseBusinessAccountType(raw string) (types.AccountType, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "spot", "account_type_spot":
		return types.AccountType_ACCOUNT_TYPE_SPOT, nil
	case "funding", "account_type_funding":
		return types.AccountType_ACCOUNT_TYPE_FUNDING, nil
	default:
		return types.AccountType_ACCOUNT_TYPE_UNSPECIFIED, fmt.Errorf("account type must be spot or funding")
	}
}
