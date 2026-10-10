package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdQuerySpotResourceParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-resource-params",
		Short: "query spot resource parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotResourceParams(
				context.Background(),
				&types.QuerySpotResourceParamsRequest{},
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

func CmdQuerySpotStatefulOrders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-stateful-orders [clob_pair_id]",
		Short: "query active spot long-term order ids for one market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clobPairId, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotStatefulOrdersByClobPair(
				context.Background(),
				&types.QuerySpotStatefulOrdersByClobPairRequest{
					ClobPairId: clobPairId,
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	return cmd
}

func CmdQueryStatefulSpotReservation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stateful-spot-reservation [owner] [number] [client_id] [clob_pair_id] [order_flags]",
		Short: "query the reservation owned by one spot long-term order",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			orderId, err := orderIDFromCLIArgs(args)
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).StatefulSpotReservation(
				context.Background(),
				&types.QueryStatefulSpotReservationRequest{OrderId: orderId},
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

func CmdQuerySpotPairLifecycle() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-pair-lifecycle [clob_pair_id]",
		Short: "query pause source and final settlement state for one spot market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clobPairId, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotPairLifecycle(
				context.Background(),
				&types.QuerySpotPairLifecycleRequest{ClobPairId: clobPairId},
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

func CmdQuerySpotDelistCursor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "spot-delist-cursor [clob_pair_id]",
		Short: "query the long-term delist cursor for one spot market",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clobPairId, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).SpotDelistCursor(
				context.Background(),
				&types.QuerySpotDelistCursorRequest{ClobPairId: clobPairId},
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

func CmdQueryOrderFillAmount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "order-fill-amount [owner] [number] [client_id] [clob_pair_id] [order_flags]",
		Short: "query the current filled base quantums for one order",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			number, err := cast.ToUint32E(args[1])
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
			orderFlags, err := cast.ToUint32E(args[4])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).OrderFillAmount(
				context.Background(),
				&types.QueryOrderFillAmountRequest{
					Owner:      args[0],
					Number:     number,
					ClientId:   clientId,
					ClobPairId: clobPairId,
					OrderFlags: orderFlags,
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

func orderIDFromCLIArgs(args []string) (types.OrderId, error) {
	number, err := cast.ToUint32E(args[1])
	if err != nil {
		return types.OrderId{}, err
	}
	clientId, err := cast.ToUint32E(args[2])
	if err != nil {
		return types.OrderId{}, err
	}
	clobPairId, err := cast.ToUint32E(args[3])
	if err != nil {
		return types.OrderId{}, err
	}
	orderFlags, err := cast.ToUint32E(args[4])
	if err != nil {
		return types.OrderId{}, err
	}
	return types.OrderId{
		SubaccountId: satypes.SubaccountId{
			Owner:  args[0],
			Number: number,
		},
		ClientId:   clientId,
		ClobPairId: clobPairId,
		OrderFlags: orderFlags,
	}, nil
}
