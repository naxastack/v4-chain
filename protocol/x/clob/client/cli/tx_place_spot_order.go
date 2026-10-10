package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	customflags "github.com/dydxprotocol/v4-chain/protocol/app/flags"
	aptypes "github.com/dydxprotocol/v4-chain/protocol/x/accountplus/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdPlaceSpotOrder() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "place-spot-order owner subaccount_number clientId clobPairId side quantums subticks goodTil maxFeePpm",
		Short: "Broadcast a spot MsgPlaceOrder. Short-term by default; pass --long-term to use goodTil as good_til_block_time.",
		Args:  cobra.ExactArgs(9),
		RunE: func(cmd *cobra.Command, args []string) error {
			longTerm, err := cmd.Flags().GetBool(flagSpotLongTerm)
			if err != nil {
				return err
			}
			orderEpoch, err := cmd.Flags().GetUint64(flagSpotOrderEpoch)
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
			side, err := cast.ToInt32E(args[4])
			if err != nil {
				return err
			}
			quantums, err := cast.ToUint64E(args[5])
			if err != nil {
				return err
			}
			subticks, err := cast.ToUint64E(args[6])
			if err != nil {
				return err
			}
			goodTil, err := cast.ToUint32E(args[7])
			if err != nil {
				return err
			}
			maxFeePpm, err := cast.ToUint32E(args[8])
			if err != nil {
				return err
			}

			msg, err := newSpotPlaceOrderMsg(SpotOrderCommandArgs{
				Owner:            args[0],
				SubaccountNumber: subaccountNumber,
				ClientId:         clientId,
				ClobPairId:       clobPairId,
				Side:             side,
				Quantums:         quantums,
				Subticks:         subticks,
				GoodTil:          goodTil,
				MaxTradingFeePpm: maxFeePpm,
				OrderEpoch:       orderEpoch,
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
	cmd.Flags().Bool(flagSpotLongTerm, false, "submit a long-term spot order; goodTil is unix good_til_block_time")
	cmd.Flags().Uint64(flagSpotOrderEpoch, 0, "payment-asset epoch for a short-term spot order")
	return cmd
}

func broadcastCliTx(cmd *cobra.Command, msg sdkMsg) error {
	clientCtx, err := client.GetClientTxContext(cmd)
	if err != nil {
		return err
	}
	txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
	if err != nil {
		return err
	}
	authenticatorIds, err := customflags.GetPermisionedKeyAuthenticatorsForExtOptions(cmd)
	if err == nil && len(authenticatorIds) > 0 {
		value, err := codectypes.NewAnyWithValue(
			&aptypes.TxExtension{SelectedAuthenticators: authenticatorIds},
		)
		if err != nil {
			return err
		}
		txf = txf.WithNonCriticalExtensionOptions(value)
	}
	return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
}
