package keeper_test

import (
	"math/big"
	"testing"

	"github.com/dydxprotocol/v4-chain/protocol/dtypes"
	"github.com/dydxprotocol/v4-chain/protocol/mocks"
	"github.com/dydxprotocol/v4-chain/protocol/testutil/constants"
	keepertest "github.com/dydxprotocol/v4-chain/protocol/testutil/keeper"
	sampletest "github.com/dydxprotocol/v4-chain/protocol/testutil/sample"
	assettypes "github.com/dydxprotocol/v4-chain/protocol/x/assets/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/memclob"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	feetierstypes "github.com/dydxprotocol/v4-chain/protocol/x/feetiers/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func configureSpotActivationParams(t *testing.T, ks keepertest.ClobKeepersTestContext) {
	require.NoError(t, ks.ClobKeeper.SetSpotResourceParams(ks.Ctx, types.DefaultSpotResourceParams()))
	require.NoError(t, ks.FeeTiersKeeper.SetSpotFeeParams(ks.Ctx, feetierstypes.DefaultSpotFeeParams()))
}

func createActiveSpotClobPair(t *testing.T, ks keepertest.ClobKeepersTestContext, pair types.ClobPair) {
	t.Helper()
	active := pair
	initial := pair
	initial.Status = types.ClobPair_STATUS_CANCEL_ONLY
	_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, initial)
	require.NoError(t, err)
	active.Status = types.ClobPair_STATUS_ACTIVE
	require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, active))
}

func TestCreateSpotClobPairValidatesAssetPolicies(t *testing.T) {
	newContext := func(t *testing.T) keepertest.ClobKeepersTestContext {
		indexer := &mocks.IndexerEventManager{}
		indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
		ks := keepertest.NewClobKeepersTestContext(
			t,
			memclob.NewMemClobPriceTimePriority(false),
			&mocks.BankKeeper{},
			indexer,
		)
		require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
		_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
		require.NoError(t, err)
		for _, assetId := range []uint32{0, 1} {
			require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
				AssetId:            assetId,
				Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
				WithdrawalsEnabled: true,
				SpotTradingEnabled: true,
			}))
		}
		return ks
	}

	pair := types.ClobPair{
		Id:                        100,
		Metadata:                  &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 0}},
		StepBaseQuantums:          10,
		QuantumConversionExponent: 0,
		SubticksPerTick:           1,
		Status:                    types.ClobPair_STATUS_CANCEL_ONLY,
	}

	t.Run("active policies", func(t *testing.T) {
		ks := newContext(t)
		created, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
		require.NoError(t, err)
		require.Equal(t, pair, created)
		stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
		require.True(t, found)
		require.Equal(t, pair, stored)
	})
	t.Run("active market must use activation transition", func(t *testing.T) {
		ks := newContext(t)
		active := pair
		active.Status = types.ClobPair_STATUS_ACTIVE
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, active)
		require.ErrorIs(t, err, types.ErrInvalidClobPairParameter)
	})

	t.Run("non-USDC quote asset", func(t *testing.T) {
		ks := newContext(t)
		invalid := pair
		invalid.Metadata = &types.ClobPair_SpotClobMetadata{
			SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 1, QuoteAssetId: 2},
		}
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, invalid)
		require.ErrorContains(t, err, "must use asset 0 as quote asset")
	})
	t.Run("missing asset", func(t *testing.T) {
		ks := newContext(t)
		invalid := pair
		invalid.Metadata = &types.ClobPair_SpotClobMetadata{
			SpotClobMetadata: &types.SpotClobMetadata{BaseAssetId: 2, QuoteAssetId: 0},
		}
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, invalid)
		require.ErrorContains(t, err, "Asset does not exist")
	})

	t.Run("trading disabled", func(t *testing.T) {
		ks := newContext(t)
		require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId:            1,
			Status:             assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
			WithdrawalsEnabled: true,
			SpotTradingEnabled: true,
		}))
		_, err := ks.ClobKeeper.CreateSpotClobPair(ks.Ctx, pair)
		require.ErrorContains(t, err, "Asset spot trading is disabled")
	})
}

func TestValidateSpotMarketActivation(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	configureSpotActivationParams(t, ks)
	pair := types.ClobPair{
		Id: 200,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
	}
	require.NoError(t, ks.ClobKeeper.ValidateSpotMarketActivation(ks.Ctx, pair))

	paused := pair
	paused.Status = types.ClobPair_STATUS_CANCEL_ONLY
	require.ErrorIs(t, ks.ClobKeeper.ValidateSpotMarketActivation(ks.Ctx, paused), types.ErrInvalidClobPairStatusTransition)

	require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
		AssetId: 1, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
		WithdrawalsEnabled: true, SpotTradingEnabled: true,
	}))
	require.ErrorContains(t, ks.ClobKeeper.ValidateSpotMarketActivation(ks.Ctx, pair), "Asset spot trading is disabled")
}

func TestUpdateSpotClobPairLifecycleTransitions(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	configureSpotActivationParams(t, ks)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	pair := types.ClobPair{
		Id: 100,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10, QuantumConversionExponent: 0, SubticksPerTick: 1,
		Status: types.ClobPair_STATUS_ACTIVE,
	}
	createActiveSpotClobPair(t, ks, pair)
	lifecycleState, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.SpotPairLifecycleState{}, lifecycleState)

	for _, status := range []types.ClobPair_Status{
		types.ClobPair_STATUS_CANCEL_ONLY,
		types.ClobPair_STATUS_ACTIVE,
		types.ClobPair_STATUS_FINAL_SETTLEMENT,
	} {
		pair.Status = status
		require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
		stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
		require.True(t, found)
		require.Equal(t, status, stored.Status)
		lifecycleState, found = ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, types.ClobPairId(pair.Id))
		require.True(t, found)
		switch status {
		case types.ClobPair_STATUS_CANCEL_ONLY:
			require.Equal(t, types.SpotPauseSourceManual, lifecycleState.PauseSource)
			require.False(t, lifecycleState.FinalSettlementStarted)
		case types.ClobPair_STATUS_ACTIVE:
			require.Equal(t, types.SpotPauseSourceNone, lifecycleState.PauseSource)
			require.False(t, lifecycleState.FinalSettlementStarted)
		case types.ClobPair_STATUS_FINAL_SETTLEMENT:
			require.True(t, lifecycleState.FinalSettlementStarted)
		}
	}

	pair.Status = types.ClobPair_STATUS_ACTIVE
	require.ErrorIs(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair), types.ErrInvalidClobPairStatusTransition)
}

func TestAdvanceSpotAssetLifecyclePreservesManualPause(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(t, memclob.NewMemClobPriceTimePriority(false), &mocks.BankKeeper{}, indexer)
	configureSpotActivationParams(t, ks)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	pair := types.ClobPair{
		Id: 101,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
	}
	createActiveSpotClobPair(t, ks, pair)

	require.NoError(t, ks.ClobKeeper.AdvanceSpotAssetLifecycle(
		ks.Ctx, types.ClobPairId(pair.Id), types.ClobPair_STATUS_CANCEL_ONLY,
	))
	state, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceAuto, state.PauseSource)
	require.NoError(t, ks.ClobKeeper.AdvanceSpotAssetLifecycle(
		ks.Ctx, types.ClobPairId(pair.Id), types.ClobPair_STATUS_ACTIVE,
	))

	pair.Status = types.ClobPair_STATUS_CANCEL_ONLY
	require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
	state, found = ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceManual, state.PauseSource)
	require.NoError(t, ks.ClobKeeper.AdvanceSpotAssetLifecycle(
		ks.Ctx, types.ClobPairId(pair.Id), types.ClobPair_STATUS_ACTIVE,
	))
	stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, stored.Status)
}

func TestAdvanceSpotAssetLifecycleRunsFullActivationGate(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(
		t,
		memclob.NewMemClobPriceTimePriority(false),
		&mocks.BankKeeper{},
		indexer,
	)
	configureSpotActivationParams(t, ks)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	pair := types.ClobPair{
		Id: 103,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
	}
	createActiveSpotClobPair(t, ks, pair)
	require.NoError(t, ks.ClobKeeper.AdvanceSpotAssetLifecycle(
		ks.Ctx, types.ClobPairId(pair.Id), types.ClobPair_STATUS_CANCEL_ONLY,
	))

	// Simulate invalid persisted activation parameters. The automatic recovery
	// path must run the same complete gate as a governance recovery.
	invalidParams := types.SpotResourceParams{}
	ks.Ctx.KVStore(ks.StoreKey).Set(
		[]byte(types.SpotResourceParamsKey),
		ks.Cdc.MustMarshal(&invalidParams),
	)
	err = ks.ClobKeeper.AdvanceSpotAssetLifecycle(
		ks.Ctx, types.ClobPairId(pair.Id), types.ClobPair_STATUS_ACTIVE,
	)
	require.ErrorContains(t, err, "invalid spot resource params")
	stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, stored.Status)
	state, found := ks.ClobKeeper.GetSpotPairLifecycleState(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.SpotPauseSourceAuto, state.PauseSource)
}

func TestSpotMarketLifecycleDoesNotBlockExitToFunding(t *testing.T) {
	for _, status := range []types.ClobPair_Status{
		types.ClobPair_STATUS_CANCEL_ONLY,
		types.ClobPair_STATUS_FINAL_SETTLEMENT,
	} {
		t.Run(status.String(), func(t *testing.T) {
			indexer := &mocks.IndexerEventManager{}
			indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			ks := keepertest.NewClobKeepersTestContext(
				t,
				memclob.NewMemClobPriceTimePriority(false),
				&mocks.BankKeeper{},
				indexer,
			)
			configureSpotActivationParams(t, ks)
			require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
			require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
				AssetId: assettypes.AssetUsdc.Id, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
				DepositsEnabled: true, WithdrawalsEnabled: true, SpotTradingEnabled: true,
			}))
			_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
			require.NoError(t, err)
			require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
				AssetId: 1, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
				DepositsEnabled: true, WithdrawalsEnabled: true, SpotTradingEnabled: true,
			}))

			pair := types.ClobPair{
				Id: 104,
				Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
					BaseAssetId: 1, QuoteAssetId: assettypes.AssetUsdc.Id,
				}},
				StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
			}
			initialPair := pair
			initialPair.Status = types.ClobPair_STATUS_CANCEL_ONLY
			_, err = ks.ClobKeeper.CreateSpotClobPairAndMemStructs(ks.Ctx, initialPair)
			require.NoError(t, err)
			require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
			pair.Status = status
			require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))

			owner := sampletest.AccAddress()
			spotId := satypes.SubaccountId{Owner: owner, Number: 1}
			fundingId := satypes.SubaccountId{Owner: owner, Number: 2}
			ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
				Id: &spotId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
				AssetPositions: []*satypes.AssetPosition{{
					AssetId: assettypes.AssetUsdc.Id, Quantums: dtypes.NewInt(100),
					StatefulReservedQuantums: dtypes.ZeroInt(),
				}},
			})
			ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
				Id: &fundingId, AccountType: satypes.AccountType_ACCOUNT_TYPE_FUNDING,
			})
			ks.SubaccountsKeeper.SetTypedSubaccount(
				ks.Ctx, owner, satypes.AccountType_ACCOUNT_TYPE_SPOT, spotId,
			)
			ks.SubaccountsKeeper.SetTypedSubaccount(
				ks.Ctx, owner, satypes.AccountType_ACCOUNT_TYPE_FUNDING, fundingId,
			)

			require.NoError(t, ks.SubaccountsKeeper.TransferFundsFromSubaccountToSubaccount(
				ks.Ctx, spotId, fundingId, assettypes.AssetUsdc.Id, big.NewInt(100),
			))
			require.Empty(t, ks.SubaccountsKeeper.GetSubaccount(ks.Ctx, spotId).AssetPositions)
			funding := ks.SubaccountsKeeper.GetSubaccount(ks.Ctx, fundingId)
			require.Len(t, funding.AssetPositions, 1)
			require.Equal(t, int64(100), funding.AssetPositions[0].Quantums.BigInt().Int64())
		})
	}
}

func TestReplaySpotShortTermOrderDoesNotReenterInactiveMarket(t *testing.T) {
	for _, status := range []types.ClobPair_Status{
		types.ClobPair_STATUS_CANCEL_ONLY,
		types.ClobPair_STATUS_FINAL_SETTLEMENT,
	} {
		t.Run(status.String(), func(t *testing.T) {
			indexer := &mocks.IndexerEventManager{}
			indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
			memoryClob := memclob.NewMemClobPriceTimePriority(false)
			ks := keepertest.NewClobKeepersTestContext(t, memoryClob, &mocks.BankKeeper{}, indexer)
			configureSpotActivationParams(t, ks)
			require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
			_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
			require.NoError(t, err)
			for _, assetId := range []uint32{0, 1} {
				require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
					AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
					DepositsEnabled: true, WithdrawalsEnabled: true, SpotTradingEnabled: true,
				}))
			}
			pair := types.ClobPair{
				Id: 105,
				Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
					BaseAssetId: 1, QuoteAssetId: assettypes.AssetUsdc.Id,
				}},
				StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
			}
			initialPair := pair
			initialPair.Status = types.ClobPair_STATUS_CANCEL_ONLY
			_, err = ks.ClobKeeper.CreateSpotClobPairAndMemStructs(ks.Ctx, initialPair)
			require.NoError(t, err)
			require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
			pair.Status = status
			require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))

			spotId := satypes.SubaccountId{Owner: sampletest.AccAddress(), Number: 1}
			ks.SubaccountsKeeper.SetSubaccount(ks.Ctx, satypes.Subaccount{
				Id: &spotId, AccountType: satypes.AccountType_ACCOUNT_TYPE_SPOT,
				AssetPositions: []*satypes.AssetPosition{{
					AssetId: assettypes.AssetUsdc.Id, Quantums: dtypes.NewInt(100),
					StatefulReservedQuantums: dtypes.ZeroInt(),
				}},
			})
			ks.SubaccountsKeeper.SetTypedSubaccount(
				ks.Ctx, spotId.Owner, satypes.AccountType_ACCOUNT_TYPE_SPOT, spotId,
			)
			order := types.Order{
				OrderId: types.OrderId{
					SubaccountId: spotId,
					ClientId:     1, ClobPairId: pair.Id,
				},
				Side: types.Order_SIDE_BUY, Quantums: 10, Subticks: 2,
				GoodTilOneof:     &types.Order_GoodTilBlock{GoodTilBlock: 20},
				MaxTradingFeePpm: 1_000,
			}
			msg := types.NewMsgPlaceOrder(order)
			txBuilder := constants.TestEncodingCfg.TxConfig.NewTxBuilder()
			require.NoError(t, txBuilder.SetMsgs(msg))
			txBytes, err := constants.TestEncodingCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
			require.NoError(t, err)
			ctx := ks.Ctx.WithIsCheckTx(true).WithBlockHeight(1).WithTxBytes(txBytes)

			_, _, _, err = ks.ClobKeeper.ReplayPlaceOrder(ctx, msg)
			require.ErrorIs(t, err, types.ErrOrderConflictsWithClobPairStatus)
			_, found := memoryClob.GetOrder(order.OrderId)
			require.False(t, found)
			operations, _ := memoryClob.GetOperationsToReplay(ctx)
			require.Empty(t, operations)
		})
	}
}

func TestUpdateSpotClobPairCanPauseWithDisabledAssetButCannotReactivate(t *testing.T) {
	indexer := &mocks.IndexerEventManager{}
	indexer.On("AddTxnEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()
	ks := keepertest.NewClobKeepersTestContext(t, memclob.NewMemClobPriceTimePriority(false), &mocks.BankKeeper{}, indexer)
	configureSpotActivationParams(t, ks)
	require.NoError(t, keepertest.CreateUsdcAsset(ks.Ctx, ks.AssetsKeeper))
	_, err := ks.AssetsKeeper.CreateAsset(ks.Ctx, 1, "BASE", "ubase", 0, false, 0, 0)
	require.NoError(t, err)
	for _, assetId := range []uint32{0, 1} {
		require.NoError(t, ks.AssetsKeeper.CreateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
			AssetId: assetId, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_ACTIVE,
			WithdrawalsEnabled: true, SpotTradingEnabled: true,
		}))
	}
	pair := types.ClobPair{
		Id: 102,
		Metadata: &types.ClobPair_SpotClobMetadata{SpotClobMetadata: &types.SpotClobMetadata{
			BaseAssetId: 1, QuoteAssetId: 0,
		}},
		StepBaseQuantums: 10, SubticksPerTick: 1, Status: types.ClobPair_STATUS_ACTIVE,
	}
	createActiveSpotClobPair(t, ks, pair)
	require.NoError(t, ks.AssetsKeeper.UpdateAssetPolicy(ks.Ctx, assettypes.AssetPolicy{
		AssetId: 1, Status: assettypes.AssetPolicyStatus_ASSET_POLICY_STATUS_PAUSED,
		WithdrawalsEnabled: true, SpotTradingEnabled: true,
	}))

	pair.Status = types.ClobPair_STATUS_CANCEL_ONLY
	require.NoError(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
	pair.Status = types.ClobPair_STATUS_ACTIVE
	require.Error(t, ks.ClobKeeper.UpdateClobPair(ks.Ctx, pair))
	stored, found := ks.ClobKeeper.GetClobPair(ks.Ctx, types.ClobPairId(pair.Id))
	require.True(t, found)
	require.Equal(t, types.ClobPair_STATUS_CANCEL_ONLY, stored.Status)
}
