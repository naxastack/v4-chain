package cli

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
)

func CmdQueryAssetPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-policy [asset_id]",
		Short: "query one asset policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assetID, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).AssetPolicy(
				context.Background(),
				&types.QueryAssetPolicyRequest{AssetId: assetID},
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

func CmdQueryAssetPolicyState() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-policy-state [asset_id]",
		Short: "query one asset policy version state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assetID, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).AssetPolicyState(
				context.Background(),
				&types.QueryAssetPolicyStateRequest{AssetId: assetID},
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

func CmdQueryAssetLifecycleTask() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-lifecycle-task [asset_id]",
		Short: "query one asset lifecycle task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			assetID, err := cast.ToUint32E(args[0])
			if err != nil {
				return err
			}
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).AssetLifecycleTask(
				context.Background(),
				&types.QueryAssetLifecycleTaskRequest{AssetId: assetID},
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

func CmdQueryAssetLifecycleSchedulerCursor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset-lifecycle-scheduler-cursor",
		Short: "query the asset lifecycle scheduler cursor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			res, err := types.NewQueryClient(clientCtx).AssetLifecycleSchedulerCursor(
				context.Background(),
				&types.QueryAssetLifecycleSchedulerCursorRequest{},
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
