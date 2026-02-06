package testnet

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
)

var _ changeset.RegistryProvider = (*PipelinesRegistryProvider)(nil)

// PipelinesRegistryProvider wraps the changeset.BaseRegistryProvider and
// provides an Init function to register Pipelines with the underlying registry.
type PipelinesRegistryProvider struct {
	*changeset.BaseRegistryProvider
}

// NewPipelinesRegistryProvider creates a new PipelinesRegistryProvider.
// The BaseRegistryProvider is initialized with a default registry.
func NewPipelinesRegistryProvider() *PipelinesRegistryProvider {
	return &PipelinesRegistryProvider{
		BaseRegistryProvider: changeset.NewBaseRegistryProvider(),
	}
}

// Archive is a noop for the DurablePipelines.
func (*PipelinesRegistryProvider) Archive() {}

// Init is used to register Durable Pipelines with the registry
//
// Add your Durable Pipeline changeset to the registry here.
func (p *PipelinesRegistryProvider) Init() error {
	// Uncomment this line to start adding new Pipelines to the registry
	// lint:ignore Intentionally unused for future changesets
	// registry := p.Registry()

	return nil
}
