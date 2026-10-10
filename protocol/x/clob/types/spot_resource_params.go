package types

import "fmt"

const (
	DefaultMaxSpotStatefulOrdersPerSubaccount uint32 = 100
	DefaultMaxSpotDelistOrdersPerBlock        uint32 = 100
	DefaultMaxAssetLifecyclePairsPerBlock     uint32 = 10
	MaxAssetLifecyclePairsPerBlock            uint32 = 100
	MaxSpotResourceLimit                      uint32 = 1_000
)

func DefaultSpotResourceParams() SpotResourceParams {
	return SpotResourceParams{
		MaxSpotStatefulOrdersPerSubaccount: DefaultMaxSpotStatefulOrdersPerSubaccount,
		MaxSpotDelistOrdersPerBlock:        DefaultMaxSpotDelistOrdersPerBlock,
		MaxAssetLifecyclePairsPerBlock:     DefaultMaxAssetLifecyclePairsPerBlock,
	}
}

func (p SpotResourceParams) Validate() error {
	if p.MaxSpotStatefulOrdersPerSubaccount == 0 ||
		p.MaxSpotStatefulOrdersPerSubaccount > MaxSpotResourceLimit {
		return fmt.Errorf("max spot stateful orders per subaccount must be between 1 and %d", MaxSpotResourceLimit)
	}
	if p.MaxSpotDelistOrdersPerBlock == 0 ||
		p.MaxSpotDelistOrdersPerBlock > MaxSpotResourceLimit {
		return fmt.Errorf("max spot delist orders per block must be between 1 and %d", MaxSpotResourceLimit)
	}
	if p.MaxAssetLifecyclePairsPerBlock == 0 ||
		p.MaxAssetLifecyclePairsPerBlock > MaxAssetLifecyclePairsPerBlock {
		return fmt.Errorf("max asset lifecycle pairs per block must be between 1 and %d", MaxAssetLifecyclePairsPerBlock)
	}
	return nil
}
