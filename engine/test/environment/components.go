package environment

import (
	"sync"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
)

// components is a struct that contains the components of the environment.
type components struct {
	mu sync.Mutex

	Chains         []fchain.BlockChain
	Logger         logger.Logger
	catalogEnabled bool
}

// newComponents creates a new components instance.
func newComponents() *components {
	return &components{
		Chains: make([]fchain.BlockChain, 0),
		Logger: logger.Nop(),
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
