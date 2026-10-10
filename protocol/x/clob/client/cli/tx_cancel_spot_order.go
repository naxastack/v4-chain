package cli

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	customflags "github.com/dydxprotocol/v4-chain/protocol/app/flags"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdCancelSpotOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-spot-order owner subaccount_number clientId clobPairId goodTil",
		Short: "Broadcast a spot MsgCancelOrder. Short-term by default; pass --long-term to cancel a long-term order.",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			longTerm, err := cmd.Flags().GetBool(flagSpotLongTerm)
			if err != nil {
				return err
			}
			subaccountNumber, err := cast.ToUint32E(args[1])
			if err != nil {
				return err
			}
			clientId, err := cast.ToUint32E(args[2])
			if err != nil {
				return err
			}
			clobPairId, err := cast.ToUint32E(args[3])
			if err != nil {
				return err
			}
			goodTil, err := cast.ToUint32E(args[4])
			if err != nil {
				return err
			}
			msg, err := newSpotCancelOrderMsg(SpotOrderCommandArgs{
				Owner:            args[0],
				SubaccountNumber: subaccountNumber,
				ClientId:         clientId,
				ClobPairId:       clobPairId,
				GoodTil:          goodTil,
				LongTerm:         longTerm,
			})
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return broadcastCliTx(cmd, msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	customflags.AddTxPermissionedKeyFlagsToCmd(cmd)
	cmd.Flags().Bool(flagSpotLongTerm, false, "cancel a long-term spot order; goodTil is unix good_til_block_time")
	return cmd
}
