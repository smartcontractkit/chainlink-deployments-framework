package proposalanalysis

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

var _ types.ExecutionContext = &executionContext{}

type executionContext struct {
	domain          cldfdomain.Domain
	environmentName string
	blockChains     chain.BlockChains
	dataStore       datastore.DataStore
}

func (ec executionContext) Domain() cldfdomain.Domain {
	return ec.domain
}

func (ec executionContext) EnvironmentName() string {
	return ec.environmentName
}

func (ec executionContext) BlockChains() chain.BlockChains {
	return ec.blockChains
}

func (ec executionContext) DataStore() datastore.DataStore {
	return ec.dataStore
}
