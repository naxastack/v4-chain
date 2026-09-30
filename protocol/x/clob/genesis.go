package clob

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
	satypes "github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	k.InitializeForGenesis(ctx)

	// Create all `ClobPair` structs.
	for _, elem := range genState.ClobPairs {
		switch elem.Metadata.(type) {
		case *types.ClobPair_PerpetualClobMetadata:
			perpetualId, err := elem.GetPerpetualId()
			if err != nil {
				panic(errorsmod.Wrap(types.ErrInvalidClobPairParameter, err.Error()))
			}
			if _, err := k.CreatePerpetualClobPairAndMemStructs(
				ctx,
				elem.Id,
				perpetualId,
				satypes.BaseQuantums(elem.StepBaseQuantums),
				elem.QuantumConversionExponent,
				elem.SubticksPerTick,
				elem.Status,
			); err != nil {
				panic(err)
			}
		case *types.ClobPair_SpotClobMetadata:
			if _, err := k.CreateSpotClobPairAndMemStructs(ctx, elem); err != nil {
				panic(err)
			}
		default:
			panic(errorsmod.Wrap(types.ErrInvalidClobPairParameter, "unsupported CLOB pair metadata"))
		}
	}
	for _, reservation := range genState.StatefulSpotReservations {
		k.ImportStatefulSpotReservation(ctx, reservation)
	}

	// Create the `LiquidationsConfig` in state, and panic if the genesis state is invalid.
	if err := k.InitializeLiquidationsConfig(ctx, genState.LiquidationsConfig); err != nil {
		panic(err)
	}

	if err := k.InitializeBlockRateLimit(ctx, genState.BlockRateLimitConfig); err != nil {
		panic(err)
	}

	if err := k.InitializeEquityTierLimit(ctx, genState.EquityTierLimitConfig); err != nil {
		panic(err)
	}

	if err := k.SetSpotResourceParams(ctx, genState.SpotResourceParams); err != nil {
		panic(err)
	}

	k.InitializeProcessProposerMatchesEvents(ctx)
	k.ResetAllDeliveredOrderIds(ctx)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	// Read the CLOB pairs from state.
	genesis.ClobPairs = k.GetAllClobPairs(ctx)

	genesis.StatefulSpotReservations = k.GetAllStatefulSpotReservations(ctx)

	// Read the liquidations config from state.
	genesis.LiquidationsConfig = k.GetLiquidationsConfig(ctx)

	// Read the block rate limit configuration from state.
	genesis.BlockRateLimitConfig = k.GetBlockRateLimitConfiguration(ctx)

	// Read the equity tier limit configuration from state.
	genesis.EquityTierLimitConfig = k.GetEquityTierLimitConfiguration(ctx)

	// Read the spot resource parameters from state.
	genesis.SpotResourceParams = k.GetSpotResourceParams(ctx)

	return genesis
}
