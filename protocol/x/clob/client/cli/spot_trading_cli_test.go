//go:build all || integration_test

package cli_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	comethttp "github.com/cometbft/cometbft/rpc/client/http"

	networktestutil "github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	appconstants "github.com/dydxprotocol/v4-chain/protocol/app/constants"
	appflags "github.com/dydxprotocol/v4-chain/protocol/app/flags"
	daemonflags "github.com/dydxprotocol/v4-chain/protocol/daemons/flags"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/appoptions"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/network"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	clobcli "github.com/dydxprotocol/v4-chain/protocol/x/clob/client/testutil"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	sendingcli "github.com/dydxprotocol/v4-chain/protocol/x/sending/client/testutil"
	sacli "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/client/testutil"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/require"
)

const (
	spotFlowPairID       = uint32(7)
	spotFlowBaseAssetID  = uint32(1)
	spotFlowFundingNum   = uint32(1)
	spotFlowSpotNum      = uint32(2)
	spotFlowBankQuantums = int64(1_000_000_000_000)
	spotFlowDeposit      = uint64(100_000)
	spotFlowBaseDenom    = "ubase"
)

func TestCLISpotTradingFlow(t *testing.T) {
	appOptions := appoptions.NewFakeAppOptions()
	cfg := network.DefaultConfig(&network.NetworkConfigOptions{
		AppOptions: appOptions,
		OnNewApp: func(val networktestutil.ValidatorI) {
			testval, ok := val.(networktestutil.Validator)
			require.True(t, ok)
			appOptions.Set(daemonflags.FlagPriceDaemonEnabled, false)
			appOptions.Set(daemonflags.FlagBridgeDaemonEnabled, false)
			appOptions.Set(daemonflags.FlagOracleEnabled, false)
			appOptions.Set(daemonflags.FlagMaxDaemonUnhealthySeconds, math.MaxUint32)
			appOptions.Set(appflags.GrpcAddress, testval.AppConfig.GRPC.Address)
		},
	})
	cfg.NumValidators = 2
	cfg.Mnemonics = []string{constants.AliceMnenomic, constants.BobMnenomic}
	cfg.ChainID = appconstants.AppName
	cfg.MinGasPrices = fmt.Sprintf("0%s", sdk.DefaultBondDenom)
	cfg.GenesisState[types.ModuleName] = marshalSpotClobGenesis(t, cfg)
	cfg.GenesisState[assettypes.ModuleName] = marshalSpotAssetGenesis(t, cfg)
	cfg.GenesisState[banktypes.ModuleName] = marshalSpotBankGenesis(t, cfg)

	net := network.New(t, cfg)
	_, err := net.WaitForHeight(1)
	require.NoError(t, err)

	logPath := os.Getenv("D7_CLI_LOG_DIR")
	var logFile *os.File
	if logPath != "" {
		require.NoError(t, os.MkdirAll(logPath, 0o755))
		logFile, err = os.Create(filepath.Join(logPath, "spot-cli-trading-flow.txt"))
		require.NoError(t, err)
		defer logFile.Close()
	}
	logger := flowLog{t: t, file: logFile}

	alice := net.Validators[0]
	bob := net.Validators[1]
	nodeURI := loopbackNode(alice.Ctx.Config.RPC.ListenAddress)
	logger.nodeURI = nodeURI
	aliceCtx := alice.ClientCtx.WithNodeURI(nodeURI)
	bobCtx := bob.ClientCtx.WithNodeURI(nodeURI)

	out, err := sacli.MsgCreateSubaccountExec(aliceCtx, "node0", "funding")
	logger.record("tx subaccounts create-subaccount funding --from node0", out, err)
	out, err = sacli.MsgCreateSubaccountExec(aliceCtx, "node0", "spot")
	logger.record("tx subaccounts create-subaccount spot --from node0", out, err)
	out, err = sacli.MsgCreateSubaccountExec(bobCtx, "node1", "funding")
	logger.record("tx subaccounts create-subaccount funding --from node1", out, err)
	out, err = sacli.MsgCreateSubaccountExec(bobCtx, "node1", "spot")
	logger.record("tx subaccounts create-subaccount spot --from node1", out, err)

	out, err = sendingcli.MsgDepositAssetExec(aliceCtx, constants.AliceAccAddress, constants.AliceAccAddress, spotFlowFundingNum, spotFlowDeposit, 0)
	logger.record(fmt.Sprintf("tx sending deposit-to-subaccount %s %s %d %d --asset-id 0", constants.AliceAccAddress, constants.AliceAccAddress, spotFlowFundingNum, spotFlowDeposit), out, err)
	out, err = sendingcli.MsgCreateTransferAssetExec(aliceCtx, "node0", constants.AliceAccAddress, spotFlowFundingNum, constants.AliceAccAddress, spotFlowSpotNum, spotFlowDeposit, 0)
	logger.record(fmt.Sprintf("tx sending create-transfer %s %d %s %d %d --asset-id 0 --from node0", constants.AliceAccAddress, spotFlowFundingNum, constants.AliceAccAddress, spotFlowSpotNum, spotFlowDeposit), out, err)
	out, err = sendingcli.MsgDepositAssetExec(bobCtx, constants.BobAccAddress, constants.BobAccAddress, spotFlowFundingNum, spotFlowDeposit, spotFlowBaseAssetID)
	logger.record(fmt.Sprintf("tx sending deposit-to-subaccount %s %s %d %d --asset-id 1", constants.BobAccAddress, constants.BobAccAddress, spotFlowFundingNum, spotFlowDeposit), out, err)
	out, err = sendingcli.MsgCreateTransferAssetExec(bobCtx, "node1", constants.BobAccAddress, spotFlowFundingNum, constants.BobAccAddress, spotFlowSpotNum, spotFlowDeposit, spotFlowBaseAssetID)
	logger.record(fmt.Sprintf("tx sending create-transfer %s %d %s %d %d --asset-id 1 --from node1", constants.BobAccAddress, spotFlowFundingNum, constants.BobAccAddress, spotFlowSpotNum, spotFlowDeposit), out, err)

	currentHeight, err := net.LatestHeight()
	require.NoError(t, err)
	goodTil := uint32(currentHeight) + types.ShortBlockWindow
	out, err = clobcli.MsgPlaceSpotOrderExec(aliceCtx, "node0", constants.AliceAccAddress, spotFlowSpotNum, 1, spotFlowPairID, types.Order_SIDE_BUY, 10, 2, goodTil, 1_000, 0)
	logger.record(fmt.Sprintf("tx clob place-spot-order %s %d 1 %d 1 10 2 %d 1000 --order-epoch 0 --from node0", constants.AliceAccAddress, spotFlowSpotNum, spotFlowPairID, goodTil), out, err)
	out, err = clobcli.MsgPlaceSpotOrderExec(bobCtx, "node1", constants.BobAccAddress, spotFlowSpotNum, 2, spotFlowPairID, types.Order_SIDE_SELL, 20, 2, goodTil, 1_000, 0)
	logger.record(fmt.Sprintf("tx clob place-spot-order %s %d 2 %d 2 20 2 %d 1000 --order-epoch 0 --from node1", constants.BobAccAddress, spotFlowSpotNum, spotFlowPairID, goodTil), out, err)

	currentHeight, err = net.LatestHeight()
	require.NoError(t, err)
	_, err = net.WaitForHeight(currentHeight + 3)
	require.NoError(t, err)

	buyerFillOut, buyerFillErr := clobcli.MsgQueryOrderFillAmountExec(aliceCtx, constants.AliceAccAddress, spotFlowSpotNum, 1, spotFlowPairID, types.OrderIdFlags_ShortTerm)
	logger.record("query clob order-fill-amount alice spot 1 pair short-term", buyerFillOut, buyerFillErr)
	sellerFillOut, sellerFillErr := clobcli.MsgQueryOrderFillAmountExec(aliceCtx, constants.BobAccAddress, spotFlowSpotNum, 2, spotFlowPairID, types.OrderIdFlags_ShortTerm)
	logger.record("query clob order-fill-amount bob spot 2 pair short-term", sellerFillOut, sellerFillErr)

	var buyerFill types.QueryOrderFillAmountResponse
	require.NoError(t, cfg.Codec.UnmarshalJSON(buyerFillOut.Bytes(), &buyerFill))
	var sellerFill types.QueryOrderFillAmountResponse
	require.NoError(t, cfg.Codec.UnmarshalJSON(sellerFillOut.Bytes(), &sellerFill))
	require.Equal(t, uint64(10), buyerFill.FillAmount)
	require.Equal(t, uint64(10), sellerFill.FillAmount)

	buyerAccountOut, buyerAccountErr := sacli.MsgQuerySpotSubaccountExec(aliceCtx, constants.AliceAccAddress, spotFlowSpotNum)
	logger.record("query subaccounts show-subaccount alice spot", buyerAccountOut, buyerAccountErr)
	sellerAccountOut, sellerAccountErr := sacli.MsgQuerySpotSubaccountExec(aliceCtx, constants.BobAccAddress, spotFlowSpotNum)
	logger.record("query subaccounts show-subaccount bob spot", sellerAccountOut, sellerAccountErr)
	var buyerAccount satypes.QuerySubaccountResponse
	require.NoError(t, cfg.Codec.UnmarshalJSON(buyerAccountOut.Bytes(), &buyerAccount))
	var sellerAccount satypes.QuerySubaccountResponse
	require.NoError(t, cfg.Codec.UnmarshalJSON(sellerAccountOut.Bytes(), &sellerAccount))
	require.GreaterOrEqual(t, assetQuantums(buyerAccount.Subaccount, spotFlowBaseAssetID), int64(10))
	require.Positive(t, assetQuantums(sellerAccount.Subaccount, assettypes.AssetUsdc.Id))

	currentHeight, err = net.LatestHeight()
	require.NoError(t, err)
	cancelTil := uint32(currentHeight) + types.ShortBlockWindow
	out, err = clobcli.MsgCancelSpotOrderExec(bobCtx, "node1", constants.BobAccAddress, spotFlowSpotNum, 2, spotFlowPairID, cancelTil)
	logger.record(fmt.Sprintf("tx clob cancel-spot-order %s %d 2 %d %d --from node1", constants.BobAccAddress, spotFlowSpotNum, spotFlowPairID, cancelTil), out, err)
	_, err = net.WaitForHeight(currentHeight + 2)
	require.NoError(t, err)
	sellerFillAfterCancel, sellerFillAfterCancelErr := clobcli.MsgQueryOrderFillAmountExec(aliceCtx, constants.BobAccAddress, spotFlowSpotNum, 2, spotFlowPairID, types.OrderIdFlags_ShortTerm)
	logger.record("query clob order-fill-amount bob after cancel", sellerFillAfterCancel, sellerFillAfterCancelErr)
	require.NoError(t, cfg.Codec.UnmarshalJSON(sellerFillAfterCancel.Bytes(), &sellerFill))
	require.Equal(t, uint64(10), sellerFill.FillAmount)
}

func (l flowLog) record(command string, output interface{ Bytes() []byte }, execErr error) {
	l.t.Helper()
	text := []byte(nil)
	if output != nil {
		text = output.Bytes()
	}
	require.NoError(l.t, execErr, "%s\n%s", command, text)
	if l.file != nil {
		_, _ = fmt.Fprintf(l.file, "COMMAND: %s\nOUTPUT:\n%s\n\n", command, text)
	}
	if code, ok := broadcastCode(text); ok {
		require.Zero(l.t, code, "%s\n%s", command, text)
		if !strings.Contains(command, "place-spot-order") && !strings.Contains(command, "cancel-spot-order") {
			l.confirmIncluded(text)
		}
	}
}

type flowLog struct {
	t       *testing.T
	file    *os.File
	nodeURI string
}

func (l flowLog) confirmIncluded(text []byte) {
	l.t.Helper()
	var resp struct {
		TxHash string `json:"txhash"`
	}
	require.NoError(l.t, json.Unmarshal(text, &resp))
	hash, err := hex.DecodeString(resp.TxHash)
	require.NoError(l.t, err)
	node, err := comethttp.New(strings.Replace(l.nodeURI, "tcp://", "http://", 1), "/websocket")
	require.NoError(l.t, err)
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		res, err := node.Tx(context.Background(), hash, false)
		if err == nil && res != nil {
			require.Zerof(l.t, res.TxResult.Code, "tx %s failed in block: %s", resp.TxHash, res.TxResult.Log)
			if l.file != nil {
				_, _ = fmt.Fprintf(l.file, "INCLUDED %s height=%d code=%d log=%s\n\n", resp.TxHash, res.Height, res.TxResult.Code, res.TxResult.Log)
			}
			return
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	require.Failf(l.t, "tx not included", "%s: %v", resp.TxHash, lastErr)
}

func broadcastCode(text []byte) (uint32, bool) {
	var parsed struct {
		Code   uint32 `json:"code"`
		TxHash string `json:"txhash"`
	}
	if err := json.Unmarshal(text, &parsed); err != nil || parsed.TxHash == "" {
		return 0, false
	}
	return parsed.Code, true
}

func loopbackNode(listenAddress string) string {
	return strings.Replace(listenAddress, "0.0.0.0", "127.0.0.1", 1)
}

func assetQuantums(subaccount satypes.Subaccount, assetID uint32) int64 {
	for _, position := range subaccount.AssetPositions {
		if position.AssetId == assetID {
			return position.GetBigQuantums().Int64()
		}
	}
	return 0
}

func marshalSpotClobGenesis(t *testing.T, cfg network.Config) []byte {
	t.Helper()
	state := types.GenesisState{
		ClobPairs: []types.ClobPair{{
			Id: spotFlowPairID,
			Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
				BaseAssetId: spotFlowBaseAssetID, QuoteAssetId: assettypes.AssetUsdc.Id,
			}},
			StepBaseQuantums:          5,
			SubticksPerTick:           1,
			QuantumConversionExponent: 0,
			Status:                    types.ClobPair_STATUS_ACTIVE,
		}},
		LiquidationsConfig: types.LiquidationsConfig_Default,
		SpotResourceParams: types.DefaultSpotResourceParams(),
	}
	buf, err := cfg.Codec.MarshalJSON(&state)
	require.NoError(t, err)
	return buf
}

func marshalSpotAssetGenesis(t *testing.T, cfg network.Config) []byte {
	t.Helper()
	state := assettypes.DefaultGenesis()
	state.Assets = append(state.Assets, assettypes.Asset{
		Id:               spotFlowBaseAssetID,
		Symbol:           "BASE",
		Denom:            spotFlowBaseDenom,
		DenomExponent:    0,
		HasMarket:        false,
		AtomicResolution: 0,
	})
	state.AssetPolicies = append(state.AssetPolicies, assettypes.AssetPolicy{
		AssetId:            spotFlowBaseAssetID,
		Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
		DepositsEnabled:    true,
		WithdrawalsEnabled: true,
		SpotTradingEnabled: true,
	})
	state.AssetPolicyStates = append(state.AssetPolicyStates, assettypes.AssetPolicyState{
		AssetId:          spotFlowBaseAssetID,
		PolicyVersion:    1,
		LifecycleVersion: 1,
	})
	buf, err := cfg.Codec.MarshalJSON(state)
	require.NoError(t, err)
	return buf
}

func marshalSpotBankGenesis(t *testing.T, cfg network.Config) []byte {
	t.Helper()
	// Stake stays on the validator balances created by the test network.
	// Adding it again makes the bank supply invariant fail for the same addresses.
	userCoins := sdk.NewCoins(
		sdk.NewInt64Coin(assettypes.AssetUsdc.Denom, spotFlowBankQuantums),
		sdk.NewInt64Coin(spotFlowBaseDenom, spotFlowBankQuantums),
	)
	state := banktypes.GenesisState{
		Balances: []banktypes.Balance{
			{Address: constants.AliceAccAddress.String(), Coins: userCoins},
			{Address: constants.BobAccAddress.String(), Coins: userCoins},
		},
	}
	buf, err := cfg.Codec.MarshalJSON(&state)
	require.NoError(t, err)
	return buf
}
