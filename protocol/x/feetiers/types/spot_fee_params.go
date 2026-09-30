package types

import "fmt"

const (
	DefaultSpotTradingFeePpm uint32 = 1_000
	MaxSpotTradingFeePpm     uint32 = 1_000_000
)

func DefaultSpotFeeParams() SpotFeeParams {
	return SpotFeeParams{TradingFeePpm: DefaultSpotTradingFeePpm}
}

func (m SpotFeeParams) Validate() error {
	if m.TradingFeePpm == 0 || m.TradingFeePpm > MaxSpotTradingFeePpm {
		return fmt.Errorf("spot trading fee ppm must be between 1 and %d", MaxSpotTradingFeePpm)
	}
	return nil
}
