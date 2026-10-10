package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdQuerySpotOrderEpoch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-order-epoch [owner] [number] [asset_id]",
		Short: "query the current spot short-term order epoch for one asset",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := cast.ToUint32E(args[1])
			if err != nil {
				return err
			}
			assetId, err := cast.ToUint32E(args[2])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotOrderEpoch(
				context.Background(),
				&types.QuerySpotOrderEpochRequest{
					Owner:   args[0],
					Number:  number,
					AssetId: assetId,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
