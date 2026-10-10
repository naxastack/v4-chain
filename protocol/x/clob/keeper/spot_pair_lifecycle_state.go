package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dydxprotocol/v4-chain/protocol/lib"
	"github.com/dydxprotocol/v4-chain/protocol/x/clob/types"
)

const spotPairLifecycleStateEncodingLength = 2

func (k Keeper) GetSpotPairLifecycleState(
	ctx sdk.Context,
	clobPairId types.ClobPairId,
) (types.SpotPairLifecycleState, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotPairLifecycleStateKeyPrefix))
	value := store.Get(lib.Uint32ToKey(uint32(clobPairId)))
	state, ok := decodeSpotPairLifecycleState(value)
	if !ok {
		return types.SpotPairLifecycleState{}, false
	}
	return state, true
}

func (k Keeper) SetSpotPairLifecycleState(
	ctx sdk.Context,
	clobPairId types.ClobPairId,
	state types.SpotPairLifecycleState,
) {
	if err := state.Validate(); err != nil {
		panic(err)
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotPairLifecycleStateKeyPrefix))
	finalSettlementStarted := byte(0)
	if state.FinalSettlementStarted {
		finalSettlementStarted = 1
	}
	store.Set(
		lib.Uint32ToKey(uint32(clobPairId)),
		[]byte{byte(state.PauseSource), finalSettlementStarted},
	)
}

// AdvanceSpotAssetLifecycle is the only non-governance status transition
// entrypoint. It preserves MANUAL pauses and never applies to perpetual pairs.
func (k Keeper) AdvanceSpotAssetLifecycle(
	ctx sdk.Context,
	clobPairId types.ClobPairId,
	targetStatus types.ClobPair_Status,
) error {
	clobPair, found := k.GetClobPair(ctx, clobPairId)
	if !found || clobPair.GetSpotClobMetadata() == nil {
		return types.ErrInvalidClobPairUpdate
	}
	state, found := k.GetSpotPairLifecycleState(ctx, clobPairId)
	if !found {
		state = types.SpotPairLifecycleState{}
	}

	if targetStatus == types.ClobPair_STATUS_ACTIVE &&
		clobPair.Status == types.ClobPair_STATUS_CANCEL_ONLY &&
		state.PauseSource == types.SpotPauseSourceManual {
		return nil
	}
	if !types.IsSupportedSpotClobPairStatusTransition(clobPair.Status, targetStatus) {
		return types.ErrInvalidClobPairStatusTransition
	}

	updated := clobPair
	updated.Status = targetStatus
	if targetStatus == types.ClobPair_STATUS_ACTIVE {
		if err := k.ValidateSpotMarketActivation(ctx, updated); err != nil {
			return err
		}
	} else if err := updated.Validate(); err != nil {
		return err
	}

	switch targetStatus {
	case types.ClobPair_STATUS_ACTIVE:
		state.PauseSource = types.SpotPauseSourceNone
	case types.ClobPair_STATUS_CANCEL_ONLY:
		if clobPair.Status == types.ClobPair_STATUS_ACTIVE {
			state.PauseSource = types.SpotPauseSourceAuto
		}
	case types.ClobPair_STATUS_FINAL_SETTLEMENT:
		state.FinalSettlementStarted = true
		k.EnsureSpotDelistCursor(ctx, uint32(clobPairId))
	}
	k.SetClobPair(ctx, updated)
	k.SetSpotPairLifecycleState(ctx, clobPairId, state)
	return nil
}

// ImportSpotPairLifecycle restores one genesis lifecycle record over the
// default state written when the Spot pair is created.
func (k Keeper) ImportSpotPairLifecycle(
	ctx sdk.Context,
	lifecycle types.SpotPairLifecycle,
) error {
	if lifecycle.PauseSource > types.SpotPairPauseSource_SPOT_PAIR_PAUSE_SOURCE_MANUAL {
		return types.ErrInvalidSpotPairLifecycleState
	}
	clobPair, found := k.GetClobPair(ctx, types.ClobPairId(lifecycle.ClobPairId))
	if !found || clobPair.GetSpotClobMetadata() == nil {
		return types.ErrInvalidSpotPairLifecycleState
	}
	k.SetSpotPairLifecycleState(ctx, clobPair.GetClobPairId(), types.SpotPairLifecycleState{
		PauseSource:            types.SpotPauseSource(lifecycle.PauseSource),
		FinalSettlementStarted: lifecycle.FinalSettlementStarted,
	})
	return nil
}

// GetAllSpotPairLifecycles returns stored lifecycle records in clob pair ID order.
func (k Keeper) GetAllSpotPairLifecycles(ctx sdk.Context) []types.SpotPairLifecycle {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.SpotPairLifecycleStateKeyPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	lifecycles := make([]types.SpotPairLifecycle, 0)
	for ; iterator.Valid(); iterator.Next() {
		state, ok := decodeSpotPairLifecycleState(iterator.Value())
		if !ok {
			panic(types.ErrInvalidSpotPairLifecycleState)
		}
		lifecycles = append(lifecycles, types.SpotPairLifecycle{
			ClobPairId:             lib.BytesToUint32(iterator.Key()),
			PauseSource:            types.SpotPairPauseSource(state.PauseSource),
			FinalSettlementStarted: state.FinalSettlementStarted,
		})
	}
	return lifecycles
}

func decodeSpotPairLifecycleState(value []byte) (types.SpotPairLifecycleState, bool) {
	if len(value) != spotPairLifecycleStateEncodingLength {
		return types.SpotPairLifecycleState{}, false
	}
	return types.SpotPairLifecycleState{
		PauseSource:            types.SpotPauseSource(value[0]),
		FinalSettlementStarted: value[1] == 1,
	}, true
}
