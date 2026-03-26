package analyzer

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

var (
	_ ExecutionContext              = (*ExecutionContextNode)(nil)
	_ ParameterAnalyzerContext      = (*ParameterAnalyzerContextNode)(nil)
	_ CallAnalyzerContext           = (*CallAnalyzerContextNode)(nil)
	_ BatchOperationAnalyzerContext = (*BatchOperationAnalyzerContextNode)(nil)
)

// ExecutionContextNode is the default implementation of ExecutionContext.
type ExecutionContextNode struct {
	domain          cldfdomain.Domain
	environmentName string
	blockChains     chain.BlockChains
	dataStore       datastore.DataStore
}

// NewExecutionContextNode constructs an execution context for analyzer runs.
func NewExecutionContextNode(
	domain cldfdomain.Domain,
	environmentName string,
	blockChains chain.BlockChains,
	dataStore datastore.DataStore,
) *ExecutionContextNode {
	return &ExecutionContextNode{
		domain:          domain,
		environmentName: environmentName,
		blockChains:     blockChains,
		dataStore:       dataStore,
	}
}

func (c *ExecutionContextNode) Domain() cldfdomain.Domain {
	return c.domain
}

func (c *ExecutionContextNode) EnvironmentName() string {
	return c.environmentName
}

func (c *ExecutionContextNode) BlockChains() chain.BlockChains {
	return c.blockChains
}

func (c *ExecutionContextNode) DataStore() datastore.DataStore {
	return c.dataStore
}

// BatchOperationAnalyzerContextNode is the default implementation of BatchOperationAnalyzerContext.
type BatchOperationAnalyzerContextNode struct {
	proposal decoder.DecodedTimelockProposal
}

// NewBatchOperationAnalyzerContextNode constructs a batch operation analyzer context for analyzer runs.
func NewBatchOperationAnalyzerContextNode(
	proposal decoder.DecodedTimelockProposal,
) *BatchOperationAnalyzerContextNode {
	return &BatchOperationAnalyzerContextNode{proposal: proposal}
}

func (c *BatchOperationAnalyzerContextNode) Proposal() decoder.DecodedTimelockProposal {
	return c.proposal
}

// CallAnalyzerContextNode is the default implementation of CallAnalyzerContext.
type CallAnalyzerContextNode struct {
	proposal       decoder.DecodedTimelockProposal
	batchOperation decoder.DecodedBatchOperation
}

// NewCallAnalyzerContextNode constructs a call analyzer context for analyzer runs.
func NewCallAnalyzerContextNode(
	proposal decoder.DecodedTimelockProposal,
	batchOperation decoder.DecodedBatchOperation,
) *CallAnalyzerContextNode {
	return &CallAnalyzerContextNode{
		proposal:       proposal,
		batchOperation: batchOperation,
	}
}

func (c *CallAnalyzerContextNode) Proposal() decoder.DecodedTimelockProposal {
	return c.proposal
}

func (c *CallAnalyzerContextNode) BatchOperation() decoder.DecodedBatchOperation {
	return c.batchOperation
}

// ParameterAnalyzerContextNode is the default implementation of ParameterAnalyzerContext.
type ParameterAnalyzerContextNode struct {
	proposal       decoder.DecodedTimelockProposal
	batchOperation decoder.DecodedBatchOperation
	call           decoder.DecodedCall
}

// NewParameterAnalyzerContextNode constructs a parameter analyzer context for analyzer runs.
func NewParameterAnalyzerContextNode(
	proposal decoder.DecodedTimelockProposal,
	batchOperation decoder.DecodedBatchOperation,
	call decoder.DecodedCall,
) *ParameterAnalyzerContextNode {
	return &ParameterAnalyzerContextNode{
		proposal:       proposal,
		batchOperation: batchOperation,
		call:           call,
	}
}

func (c *ParameterAnalyzerContextNode) Proposal() decoder.DecodedTimelockProposal {
	return c.proposal
}

func (c *ParameterAnalyzerContextNode) BatchOperation() decoder.DecodedBatchOperation {
	return c.batchOperation
}

func (c *ParameterAnalyzerContextNode) Call() decoder.DecodedCall {
	return c.call
}
