package analyzer

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

type ProposalContext interface {
	GetEVMRegistry() EVMABIRegistry
	GetSolanaDecoderRegistry() SolanaDecoderRegistry
	FieldsContext(chainSelector uint64) *FieldContext
	GetRenderer() Renderer
	SetRenderer(renderer Renderer)
}

type ProposalContextProvider func(env deployment.Environment) (ProposalContext, error)

// DefaultProposalContext implements a default proposal analysis context which searches
// for the EVM ABI of all known contracts.
type DefaultProposalContext struct {
	AddressesByChain deployment.AddressesByChain
	evmRegistry      EVMABIRegistry
	solanaRegistry   SolanaDecoderRegistry
	renderer         Renderer
}

func (c *DefaultProposalContext) GetEVMRegistry() EVMABIRegistry {
	return c.evmRegistry
}

func (c *DefaultProposalContext) GetSolanaDecoderRegistry() SolanaDecoderRegistry {
	return c.solanaRegistry
}

func (c *DefaultProposalContext) GetRenderer() Renderer {
	if c.renderer == nil {
		c.renderer = NewMarkdownRenderer() // Default fallback
	}

	return c.renderer
}

func (c *DefaultProposalContext) SetRenderer(renderer Renderer) {
	c.renderer = renderer
}

type proposalCtxOption func(*proposalCtxOptions) error

type proposalCtxOptions struct {
	evmABIMappings map[string]string
	solanaDecoders map[string]DecodeInstructionFn
	renderer       Renderer
}

func WithEVMABIMappings(mappings map[string]string) proposalCtxOption {
	return func(o *proposalCtxOptions) error {
		o.evmABIMappings = mappings
		return nil
	}
}

func WithSolanaDecoders(decoders map[string]DecodeInstructionFn) proposalCtxOption {
	return func(o *proposalCtxOptions) error {
		o.solanaDecoders = decoders
		return nil
	}
}

func WithRenderer(renderer Renderer) proposalCtxOption {
	return func(o *proposalCtxOptions) error {
		o.renderer = renderer
		return nil
	}
}

func NewDefaultProposalContext(env deployment.Environment, opts ...proposalCtxOption) (ProposalContext, error) {
	// Apply options
	options := &proposalCtxOptions{
		evmABIMappings: map[string]string{},
		solanaDecoders: map[string]DecodeInstructionFn{},
	}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, err
		}
	}
	addressesByChain, errAddrBook := env.ExistingAddresses.Addresses() //nolint:staticcheck
	if errAddrBook != nil {
		return nil, errAddrBook
	}
	dataStoreAddresses, errFetch := env.DataStore.Addresses().Fetch()
	if errFetch != nil {
		return nil, errFetch
	}
	for _, address := range dataStoreAddresses {
		chainAddresses, exists := addressesByChain[address.ChainSelector]
		if !exists {
			chainAddresses = map[string]deployment.TypeAndVersion{}
		}
		chainAddresses[address.Address] = deployment.TypeAndVersion{
			Type:    deployment.ContractType(address.Type),
			Version: pointer.DerefOrEmpty(address.Version),
			Labels:  deployment.NewLabelSet(address.Labels.List()...),
		}
		addressesByChain[address.ChainSelector] = chainAddresses
	}
	// Initialize contract registries
	var evmRegistry EVMABIRegistry
	var solanaRegistry SolanaDecoderRegistry
	var err error
	if len(options.solanaDecoders) > 0 {
		solanaRegistry, err = NewEnvironmentSolanaRegistry(env, options.solanaDecoders)
		if err != nil {
			return nil, err
		}
	}
	if len(options.evmABIMappings) > 0 {
		evmRegistry, err = NewEnvironmentEVMRegistry(env, options.evmABIMappings)
		if err != nil {
			return nil, err
		}
	}

	return &DefaultProposalContext{
		evmRegistry:      evmRegistry,
		solanaRegistry:   solanaRegistry,
		AddressesByChain: addressesByChain,
		renderer:         options.renderer,
	}, nil
}

func (c *DefaultProposalContext) FieldsContext(chainSelector uint64) *FieldContext {
	chainAddresses := deployment.AddressesByChain{}
	chainAddresses[chainSelector] = c.AddressesByChain[chainSelector]

	return NewFieldContext(chainAddresses)
}
