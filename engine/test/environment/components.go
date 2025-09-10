package environment

import (
	"sync"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// components is a struct that contains the components of the environment.
type components struct {
	mu sync.Mutex

	Chains []fchain.BlockChain
}

// newComponents creates a new components instance.
func newComponents() *components {
	return &components{
		Chains: make([]fchain.BlockChain, 0),
	}
}

// AddChains adds chains to the components in a thread-safe manner.
func (c *components) AddChains(chains ...fchain.BlockChain) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, chain := range chains {
		if chain == nil {
			continue
		}

		c.Chains = append(c.Chains, chain)
	}
}
