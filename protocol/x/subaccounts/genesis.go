package subaccounts

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	indexerevents "github.com/dydxprotocol/v4-chain/protocol/indexer/events"
	"github.com/dydxprotocol/v4-chain/protocol/indexer/indexer_manager"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/keeper"
	"github.com/dydxprotocol/v4-chain/protocol/x/subaccounts/types"
)

// InitGenesis initializes the subaccounts module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.InitializeForGenesis(ctx)

	// Set all the subaccounts
	for _, elem := range genState.Subaccounts {
		k.SetSubaccount(ctx, elem)
		k.GetIndexerEventManager().AddTxnEvent(
			ctx,
			indexerevents.SubtypeSubaccountUpdate,
			indexerevents.SubaccountUpdateEventVersion,
			indexer_manager.GetBytes(
				indexerevents.NewSubaccountUpdateEvent(
					elem.Id,
					elem.PerpetualPositions,
					elem.AssetPositions,
					nil,
				),
			),
		)
	}
	for _, mapping := range genState.TypedSubaccounts {
		k.SetTypedSubaccount(ctx, mapping.Owner, mapping.AccountType, *mapping.SubaccountId)
	}
	for _, cursor := range genState.BusinessNumberCursors {
		k.SetNextBusinessSubaccountNumber(ctx, cursor.Owner, cursor.NextNumber)
	}
	for _, epoch := range genState.SpotOrderEpochs {
		k.SetSpotOrderEpoch(ctx, *epoch.SubaccountId, epoch.AssetId, epoch.Epoch)
	}
}

// ExportGenesis returns the subaccounts module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()

	genesis.Subaccounts = k.GetAllSubaccount(ctx)
	typedSubaccounts := k.GetAllTypedSubaccounts(ctx)
	if len(typedSubaccounts) > 0 {
		genesis.TypedSubaccounts = typedSubaccounts
	}
	cursors := k.GetAllNextBusinessSubaccountNumbers(ctx)
	if len(cursors) > 0 {
		genesis.BusinessNumberCursors = cursors
	}
	spotOrderEpochs := k.GetAllSpotOrderEpochs(ctx)
	if len(spotOrderEpochs) > 0 {
		genesis.SpotOrderEpochs = spotOrderEpochs
	}

	return genesis
}
