package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
	"github.com/spf13/cobra"
)

func CmdQuerySpotFeeParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-spot-fee-params",
		Short: "get the SpotFeeParams",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotFeeParams(
				context.Background(),
				&types.QuerySpotFeeParamsRequest{},
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
