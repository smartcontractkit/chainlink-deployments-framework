package proposalanalysis

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/samber/lo"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	analyzer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/formatter"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

type analyzerEngine struct {
	proposalAnalyzers       []types.ProposalAnalyzer
	batchOperationAnalyzers []types.BatchOperationAnalyzer
	callAnalyzers           []types.CallAnalyzer
	parameterAnalyzers      []types.ParameterAnalyzer

	decoder           decoder.ProposalDecoder
	formatterRegistry *formatter.FormatterRegistry
	evmRegistry       experimentalanalyzer.EVMABIRegistry
	solanaRegistry    experimentalanalyzer.SolanaDecoderRegistry
	executionContext  types.ExecutionContext // Store for formatters
}

var _ types.AnalyzerEngine = &analyzerEngine{}

// NewAnalyzerEngine creates a new analyzer engine
// Options can be provided to customize the engine behavior, such as injecting registries
func NewAnalyzerEngine(opts ...EngineOption) types.AnalyzerEngine {
	// Apply options to get configuration
	cfg := ApplyEngineOptions(opts...)

	engine := &analyzerEngine{
		decoder:           decoder.NewLegacyDecoder(),
		formatterRegistry: formatter.NewFormatterRegistry(),
		evmRegistry:       cfg.GetEVMRegistry(),
		solanaRegistry:    cfg.GetSolanaRegistry(),
	}
	return engine
}

func (ae *analyzerEngine) Run(
	ctx context.Context,
	domain cldfdomain.Domain,
	environmentName string,
	proposal *mcms.TimelockProposal,
) (types.AnalyzedProposal, error) {
	mcmsChainSelectors := slices.Sorted(maps.Keys(proposal.ChainMetadata))
	chainSelectors := lo.Map(mcmsChainSelectors, func(s mcmstypes.ChainSelector, _ int) uint64 { return uint64(s) })
	env, err := cldfenvironment.Load(ctx, domain, environmentName,
		cldfenvironment.OnlyLoadChainsFor(chainSelectors),
		// cldfenvironment.WithLogger(lggr),
		cldfenvironment.WithoutJD())
	if err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	// Decode proposal
	decodedProposal, err := ae.decoder.Decode(ctx, env, proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to decode timelock proposal: %w", err)
	}

	actx := &analyzerContext{
		evmRegistry:    ae.evmRegistry,
		solanaRegistry: ae.solanaRegistry,
	}
	ectx := executionContext{
		domain:          domain,
		environmentName: environmentName,
		blockChains:     env.BlockChains,
		dataStore:       env.DataStore,
	}

	// Store execution context for formatters
	ae.executionContext = &ectx

	analyzedProposal, err := ae.analyzeProposal(ctx, actx, ectx, decodedProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze timelock proposal: %w", err)
	}

	return analyzedProposal, nil
}

// Format writes the formatted proposal output to the provided io.Writer.
func (ae *analyzerEngine) Format(
	ctx context.Context,
	w io.Writer,
	formatterID string,
	proposal types.AnalyzedProposal,
) error {
	f, exists := ae.formatterRegistry.Get(formatterID)
	if !exists {
		return fmt.Errorf("formatter %s not registered", formatterID)
	}

	if ae.executionContext == nil {
		return fmt.Errorf("execution context not available - ensure Run() was called before Format()")
	}

	req := types.FormatterRequest{
		Domain:          ae.executionContext.Domain().String(),
		EnvironmentName: ae.executionContext.EnvironmentName(),
	}

	return f.Format(ctx, w, req, proposal)
}

func (ae *analyzerEngine) RegisterAnalyzer(baseAnalyzer types.BaseAnalyzer) error {
	if baseAnalyzer == nil {
		return fmt.Errorf("analyzer cannot be nil")
	}

	id := baseAnalyzer.ID()
	if id == "" {
		return fmt.Errorf("analyzer ID cannot be empty")
	}

	// Check for duplicate IDs across all analyzer types
	if ae.hasAnalyzerID(id) {
		return fmt.Errorf("analyzer with ID %q is already registered", id)
	}

	switch a := baseAnalyzer.(type) {
	case types.ProposalAnalyzer:
		ae.proposalAnalyzers = append(ae.proposalAnalyzers, a)
	case types.BatchOperationAnalyzer:
		ae.batchOperationAnalyzers = append(ae.batchOperationAnalyzers, a)
	case types.CallAnalyzer:
		ae.callAnalyzers = append(ae.callAnalyzers, a)
	case types.ParameterAnalyzer:
		ae.parameterAnalyzers = append(ae.parameterAnalyzers, a)
	default:
		return fmt.Errorf("unknown analyzer type")
	}

	return nil
}

// hasAnalyzerID checks if an analyzer with the given ID is already registered
func (ae *analyzerEngine) hasAnalyzerID(id string) bool {
	// Check proposal analyzers
	for _, a := range ae.proposalAnalyzers {
		if a.ID() == id {
			return true
		}
	}

	// Check batch operation analyzers
	for _, a := range ae.batchOperationAnalyzers {
		if a.ID() == id {
			return true
		}
	}

	// Check call analyzers
	for _, a := range ae.callAnalyzers {
		if a.ID() == id {
			return true
		}
	}

	// Check parameter analyzers
	for _, a := range ae.parameterAnalyzers {
		if a.ID() == id {
			return true
		}
	}

	return false
}

func (ae *analyzerEngine) RegisterFormatter(f types.Formatter) error {
	return ae.formatterRegistry.Register(f)
}

// trackAnnotations wraps annotations with analyzer ID tracking.
// This allows annotations to be queried by analyzer ID using GetAnnotationsByAnalyzer.
func trackAnnotations(annotations types.Annotations, analyzerID string) types.Annotations {
	tracked := make(types.Annotations, 0, len(annotations))
	for _, ann := range annotations {
		tracked = append(tracked, analyzer.NewAnnotationWithAnalyzer(
			ann.Name(),
			ann.Type(),
			ann.Value(),
			analyzerID,
		))
	}
	return tracked
}

func (ae *analyzerEngine) analyzeProposal(
	ctx context.Context,
	actx *analyzerContext,
	ectx executionContext,
	decodedProposal types.DecodedTimelockProposal,
) (types.AnalyzedProposal, error) {
	proposal := &analyzedProposal{
		Annotated:       &analyzer.Annotated{},
		decodedProposal: decodedProposal,
	}
	actx.proposal = proposal

	// STEP 1: Analyze batch operations first (bottom-up approach)
	// This allows proposal analyzers to access annotations from batch operations
	batchOps := make(types.AnalyzedBatchOperations, 0)
	for _, batchOp := range decodedProposal.BatchOperations() {
		analyzedBatchOp, err := ae.analyzeBatchOperation(ctx, actx, ectx, batchOp)
		if err != nil {
			// log error but continue
			continue
		}
		batchOps = append(batchOps, analyzedBatchOp)
	}
	proposal.batchOperations = batchOps

	// STEP 2: Now run proposal analyzers
	// They can access annotations from batch operations via AnalyzerContext
	baseAnalyzers := make([]types.BaseAnalyzer, len(ae.proposalAnalyzers))
	for i, a := range ae.proposalAnalyzers {
		baseAnalyzers[i] = a
	}

	graph, err := internal.NewDependencyGraph(baseAnalyzers)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph for proposal analyzers: %w", err)
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort proposal analyzers: %w", err)
	}

	// Execute proposal analyzers in dependency order
	for _, baseAnalyzer := range sorted {
		proposalAnalyzer := baseAnalyzer.(types.ProposalAnalyzer)

		// Create analyzer request
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}

		// Check if analyzer can analyze this proposal
		if !proposalAnalyzer.CanAnalyze(ctx, req, decodedProposal) {
			continue
		}

		annotations, err := proposalAnalyzer.Analyze(ctx, req, decodedProposal)
		if err != nil {
			// log error but continue with other analyzers
			continue
		}
		// Track which analyzer created the annotations
		trackedAnnotations := trackAnnotations(annotations, proposalAnalyzer.ID())
		proposal.AddAnnotations(trackedAnnotations...)
	}

	return proposal, nil
}

func (ae *analyzerEngine) analyzeBatchOperation(
	ctx context.Context,
	actx *analyzerContext,
	ectx executionContext,
	decodedBatchOperation types.DecodedBatchOperation,
) (types.AnalyzedBatchOperation, error) {
	batchOp := &analyzedBatchOperation{
		Annotated:             &analyzer.Annotated{},
		decodedBatchOperation: decodedBatchOperation,
	}
	actx.batchOperation = batchOp

	// STEP 1: Analyze calls first (bottom-up approach)
	// This allows batch operation analyzers to access annotations from calls
	calls := make(types.AnalyzedCalls, 0)
	for _, call := range decodedBatchOperation.Calls() {
		analyzedCall, err := ae.analyzeCall(ctx, actx, ectx, call)
		if err != nil {
			// log error but continue
			continue
		}
		calls = append(calls, analyzedCall)
	}
	batchOp.calls = calls

	// STEP 2: Now run batch operation analyzers
	// They can access annotations from calls via AnalyzerContext
	baseAnalyzers := make([]types.BaseAnalyzer, len(ae.batchOperationAnalyzers))
	for i, a := range ae.batchOperationAnalyzers {
		baseAnalyzers[i] = a
	}

	graph, err := internal.NewDependencyGraph(baseAnalyzers)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph for batch operation analyzers: %w", err)
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort batch operation analyzers: %w", err)
	}

	// Execute batch operation analyzers
	for _, baseAnalyzer := range sorted {
		batchOpAnalyzer := baseAnalyzer.(types.BatchOperationAnalyzer)

		// Create analyzer request
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}

		// Check if analyzer can analyze this batch operation
		if !batchOpAnalyzer.CanAnalyze(ctx, req, decodedBatchOperation) {
			continue
		}

		annotations, err := batchOpAnalyzer.Analyze(ctx, req, decodedBatchOperation)
		if err != nil {
			// log error but continue
			continue
		}
		trackedAnnotations := trackAnnotations(annotations, batchOpAnalyzer.ID())
		batchOp.AddAnnotations(trackedAnnotations...)
	}

	return batchOp, nil
}

func (ae *analyzerEngine) analyzeCall(
	ctx context.Context,
	actx *analyzerContext,
	ectx executionContext,
	decodedCall types.DecodedCall,
) (types.AnalyzedCall, error) {
	call := &analyzedCall{
		Annotated:   &analyzer.Annotated{},
		decodedCall: decodedCall,
	}
	actx.call = call

	// STEP 1: Analyze parameters first (bottom-up approach)
	// This allows call analyzers to access annotations from parameters
	inputs := make(types.AnalyzedParameters, 0)
	for _, param := range decodedCall.Inputs() {
		analyzedParam, err := ae.analyzeParameter(ctx, actx, ectx, param)
		if err != nil {
			// log error but continue
			continue
		}
		inputs = append(inputs, analyzedParam)
	}

	outputs := make(types.AnalyzedParameters, 0)
	for _, param := range decodedCall.Outputs() {
		analyzedParam, err := ae.analyzeParameter(ctx, actx, ectx, param)
		if err != nil {
			// log error but continue
			continue
		}
		outputs = append(outputs, analyzedParam)
	}

	call.inputs = inputs
	call.outputs = outputs

	// STEP 2: Now run call analyzers
	// They can access annotations from parameters via AnalyzerContext
	baseAnalyzers := make([]types.BaseAnalyzer, len(ae.callAnalyzers))
	for i, a := range ae.callAnalyzers {
		baseAnalyzers[i] = a
	}

	graph, err := internal.NewDependencyGraph(baseAnalyzers)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph for call analyzers: %w", err)
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort call analyzers: %w", err)
	}

	// Execute call analyzers
	for _, baseAnalyzer := range sorted {
		callAnalyzer := baseAnalyzer.(types.CallAnalyzer)

		// Create analyzer request
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}

		// Check if analyzer can analyze this call
		if !callAnalyzer.CanAnalyze(ctx, req, decodedCall) {
			continue
		}

		annotations, err := callAnalyzer.Analyze(ctx, req, decodedCall)
		if err != nil {
			// log error but continue
			continue
		}
		trackedAnnotations := trackAnnotations(annotations, callAnalyzer.ID())
		call.AddAnnotations(trackedAnnotations...)
	}

	return call, nil
}

func (ae *analyzerEngine) analyzeParameter(
	ctx context.Context,
	actx *analyzerContext,
	ectx executionContext,
	decodedParameter types.DecodedParameter,
) (types.AnalyzedParameter, error) {
	param := &analyzedParameter{
		Annotated:        &analyzer.Annotated{},
		decodedParameter: decodedParameter,
	}

	// Build dependency graph for parameter analyzers
	baseAnalyzers := make([]types.BaseAnalyzer, len(ae.parameterAnalyzers))
	for i, a := range ae.parameterAnalyzers {
		baseAnalyzers[i] = a
	}

	graph, err := internal.NewDependencyGraph(baseAnalyzers)
	if err != nil {
		return nil, fmt.Errorf("failed to build dependency graph for parameter analyzers: %w", err)
	}

	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, fmt.Errorf("failed to sort parameter analyzers: %w", err)
	}

	// Execute parameter analyzers
	for _, baseAnalyzer := range sorted {
		paramAnalyzer := baseAnalyzer.(types.ParameterAnalyzer)

		// Create analyzer request
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}

		// Check if analyzer can analyze this parameter
		if !paramAnalyzer.CanAnalyze(ctx, req, decodedParameter) {
			continue
		}

		annotations, err := paramAnalyzer.Analyze(ctx, req, decodedParameter)
		if err != nil {
			// log error but continue
			continue
		}
		trackedAnnotations := trackAnnotations(annotations, paramAnalyzer.ID())
		param.AddAnnotations(trackedAnnotations...)
	}

	return param, nil
}

var _ types.AnalyzedProposal = &analyzedProposal{}

type analyzedProposal struct {
	*analyzer.Annotated
	decodedProposal types.DecodedTimelockProposal
	batchOperations types.AnalyzedBatchOperations
}

func (a analyzedProposal) BatchOperations() types.AnalyzedBatchOperations {
	return a.batchOperations
}

// ---------------------------------------------------------------------

var _ types.AnalyzedBatchOperation = &analyzedBatchOperation{}

type analyzedBatchOperation struct {
	*analyzer.Annotated
	decodedBatchOperation types.DecodedBatchOperation
	calls                 types.AnalyzedCalls
}

func (a analyzedBatchOperation) Calls() types.AnalyzedCalls {
	return a.calls
}

// ---------------------------------------------------------------------

var _ types.AnalyzedCall = &analyzedCall{}

type analyzedCall struct {
	*analyzer.Annotated
	decodedCall types.DecodedCall
	inputs      types.AnalyzedParameters
	outputs     types.AnalyzedParameters
}

func (a analyzedCall) Name() string {
	return a.decodedCall.Name()
}

func (a analyzedCall) Inputs() types.AnalyzedParameters {
	return a.inputs
}

func (a analyzedCall) Outputs() types.AnalyzedParameters {
	return a.outputs
}

// ---------------------------------------------------------------------

var _ types.AnalyzedParameter = &analyzedParameter{}

type analyzedParameter struct {
	*analyzer.Annotated
	decodedParameter types.DecodedParameter
}

func (a analyzedParameter) Name() string {
	return a.decodedParameter.Name()
}

func (a analyzedParameter) Type() string {
	return a.decodedParameter.Type()
}

func (a analyzedParameter) Value() any {
	return a.decodedParameter.Value()
}
