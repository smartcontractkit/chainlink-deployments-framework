package onchain

import (
	"slices"

	chainselectors "github.com/smartcontractkit/chain-selectors"
)

var (
	// testSelectors defines a standard list of available test selectors for each chain family.
	testSelectors = map[string][]uint64{
		// Starts the EVM test selector TEST_90000001 and limits the number of chains that can be
		// loaded to 10. This avoid conflicts with other selectors.
		chainselectors.FamilyEVM: {
			chainselectors.TEST_90000001.Selector,
			chainselectors.TEST_90000002.Selector,
			chainselectors.TEST_90000003.Selector,
			chainselectors.TEST_90000004.Selector,
			chainselectors.TEST_90000005.Selector,
			chainselectors.TEST_90000006.Selector,
			chainselectors.TEST_90000007.Selector,
			chainselectors.TEST_90000008.Selector,
			chainselectors.TEST_90000009.Selector,
			chainselectors.TEST_90000010.Selector,
		},
		chainselectors.FamilyAptos: {
			chainselectors.APTOS_LOCALNET.Selector,
		},
		chainselectors.FamilySolana: {
			chainselectors.TEST_22222222222222222222222222222222222222222222.Selector,
			chainselectors.TEST_33333333333333333333333333333333333333333333.Selector,
			chainselectors.TEST_44444444444444444444444444444444444444444444.Selector,
		},
		chainselectors.FamilyTon: {
			chainselectors.TON_LOCALNET.Selector,
		},
		chainselectors.FamilyTron: {
			chainselectors.TRON_DEVNET.Selector,
		},
		chainselectors.FamilySui: {
			chainselectors.SUI_LOCALNET.Selector,
		},
	}

	// ZKSync selectors are defined here rather than in testSelectors to avoid adding more
	// complexity to the test selectors since zksync is an implemented as an EVM chain, and we are
	// unable to differentiate between EVM and ZKSync chains in the map.
	zksyncSelectors = []uint64{
		chainselectors.TEST_90000051.Selector,
		chainselectors.TEST_90000052.Selector,
		chainselectors.TEST_90000053.Selector,
		chainselectors.TEST_90000054.Selector,
		chainselectors.TEST_90000055.Selector,
		chainselectors.TEST_90000056.Selector,
		chainselectors.TEST_90000057.Selector,
		chainselectors.TEST_90000058.Selector,
		chainselectors.TEST_90000059.Selector,
		chainselectors.TEST_90000060.Selector,
	}
)

// getTestSelectorsByFamily returns a copy of the test selectors for the given family.
func getTestSelectorsByFamily(family string) []uint64 {
	return slices.Clone(testSelectors[family])
}
