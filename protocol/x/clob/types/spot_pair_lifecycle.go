package types

// SpotPauseSource records why a Spot market is in CANCEL_ONLY. Only AUTO
// pauses may be cleared by a later asset lifecycle task.
type SpotPauseSource byte

const (
	SpotPauseSourceNone SpotPauseSource = iota
	SpotPauseSourceAuto
	SpotPauseSourceManual
)

// SpotPairLifecycleState is consensus state owned by x/clob. It is encoded as
// two stable bytes by the keeper and intentionally does not alter ClobPair.
type SpotPairLifecycleState struct {
	PauseSource            SpotPauseSource
	FinalSettlementStarted bool
}

func (s SpotPairLifecycleState) Validate() error {
	if s.PauseSource > SpotPauseSourceManual {
		return ErrInvalidSpotPairLifecycleState
	}
	return nil
}
