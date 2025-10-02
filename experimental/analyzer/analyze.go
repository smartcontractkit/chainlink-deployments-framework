package analyzer

import (
	chainsel "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/proposalutils"

	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

type ProposalContext interface {
	GetEVMRegistry() EVMABIRegistry
	GetSolanaDecoderRegistry() SolanaDecoderRegistry
	ArgumentContext(chainSelector uint64) *proposalutils.ArgumentContext
}

type ProposalContextProvider func(env deployment.Environment) (ProposalContext, error)

func DescribeTimelockProposal(ctx ProposalContext, proposal *mcms.TimelockProposal) (string, error) {
	describedBatch, err := describeBatchOperations(ctx, proposal.Operations)
	if err != nil {
		return "", err
	}

	return proposalutils.DescribeTimelockProposal(proposal, describedBatch), nil
}

func DescribeProposal(ctx ProposalContext, proposal *mcms.Proposal) (string, error) {
	describedBatch, err := describeOperations(ctx, proposal.Operations)
	if err != nil {
		return "", err
	}

	return proposalutils.DescribeProposal(proposal, describedBatch), nil
}

func describeBatchOperations(ctx ProposalContext, batches []types.BatchOperation) ([][]string, error) {
	describedBatches := make([][]string, len(batches))
	for batchIdx, batch := range batches {
		chainSel := uint64(batch.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}
		describedBatches[batchIdx] = make([]string, len(batch.Transactions))
		switch family {
		case chainsel.FamilyEVM:
			describedTxs, err := AnalyzeEVMTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.ArgumentContext(chainSel))
			}
		case chainsel.FamilySolana:
			describedTxs, err := AnalyzeSolanaTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.ArgumentContext(chainSel))
			}
		case chainsel.FamilyAptos:
			describedTxs, err := AnalyzeAptosTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.ArgumentContext(chainSel))
			}
		case chainsel.FamilySui:
			describedTxs, err := AnalyzeSuiTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, decodedCall := range describedTxs {
				describedBatches[batchIdx][callIdx] = decodedCall.Describe(ctx.ArgumentContext(chainSel))
			}
		default:
			for callIdx := range batch.Transactions {
				describedBatches[batchIdx][callIdx] = family + " transaction decoding is not supported"
			}
		}
	}

	return describedBatches, nil
}

func describeOperations(ctx ProposalContext, operations []types.Operation) ([]string, error) {
	describedOperations := make([]string, len(operations))
	for callIdx, operation := range operations {
		chainSel := uint64(operation.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}

		switch family {
		case chainsel.FamilyEVM:
			describedTransaction, err := AnalyzeEVMTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.ArgumentContext(uint64(operation.ChainSelector)))

		case chainsel.FamilySolana:
			describedTransaction, err := AnalyzeSolanaTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.ArgumentContext(uint64(operation.ChainSelector)))

		case chainsel.FamilyAptos:
			describedTransaction, err := AnalyzeAptosTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.ArgumentContext(uint64(operation.ChainSelector)))
		case chainsel.FamilySui:
			describedTransaction, err := AnalyzeSuiTransactions(ctx, uint64(operation.ChainSelector), []types.Transaction{operation.Transaction})
			if err != nil {
				return nil, err
			}
			describedOperations[callIdx] = describedTransaction[0].Describe(ctx.ArgumentContext(uint64(operation.ChainSelector)))

		default:
			describedOperations[callIdx] = family + " transaction decoding is not supported"
		}
	}

	return describedOperations, nil
}

// DefaultProposalContext implements a default proposal analysis context which searches
// for the EVM ABI of all known contracts.
type DefaultProposalContext struct {
	AddressesByChain deployment.AddressesByChain
	evmRegistry      EVMABIRegistry
	solanaRegistry   SolanaDecoderRegistry
}

func (c *DefaultProposalContext) GetEVMRegistry() EVMABIRegistry {
	return c.evmRegistry
}

func (c *DefaultProposalContext) GetSolanaDecoderRegistry() SolanaDecoderRegistry {
	return c.solanaRegistry
}

type proposalCtxOption func(*proposalCtxOptions) error

type proposalCtxOptions struct {
	evmABIMappings map[string]string
	solanaDecoders map[string]DecodeInstructionFn
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
	}, nil
}

func (c *DefaultProposalContext) ArgumentContext(chainSelector uint64) *proposalutils.ArgumentContext {
	chainAddresses := deployment.AddressesByChain{}
	chainAddresses[chainSelector] = c.AddressesByChain[chainSelector]

	return proposalutils.NewArgumentContext(chainAddresses)
}
