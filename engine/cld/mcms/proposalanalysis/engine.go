package proposalanalysis

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenvironment "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	analyzer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type analyzerEngine struct {
	proposalAnalyzers       []types.ProposalAnalyzer
	batchOperationAnalyzers []types.BatchOperationAnalyzer
	callAnalyzers           []types.CallAnalyzer
	parameterAnalyzers      []types.ParameterAnalyzer

	evmABIMappings map[string]string
	solanaDecoders map[string]experimentalanalyzer.DecodeInstructionFn

	decoder          decoder.ProposalDecoder
	rendererRegistry *renderer.RendererRegistry

	executionContext types.ExecutionContext
	logger           logger.Logger
	analyzerTimeout  time.Duration
}

var _ types.AnalyzerEngine = &analyzerEngine{}

// NewAnalyzerEngine creates a new analyzer engine
// Options can be provided to customize the engine behavior, such as injecting a logger and timeouts.
// Chain-specific EVM ABI mappings and Solana decoders are registered via RegisterEVMABIMappings and RegisterSolanaDecoders.
func NewAnalyzerEngine(opts ...EngineOption) types.AnalyzerEngine {
	// Apply options to get configuration
	cfg := ApplyEngineOptions(opts...)

	engine := &analyzerEngine{
		evmABIMappings:   make(map[string]string),
		solanaDecoders:   make(map[string]experimentalanalyzer.DecodeInstructionFn),
		decoder:          decoder.NewLegacyDecoder(),
		rendererRegistry: renderer.NewRendererRegistry(),
		logger:           cfg.GetLogger(),
		analyzerTimeout:  cfg.GetAnalyzerTimeout(),
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
		cldfenvironment.WithLogger(ae.logger),
		cldfenvironment.WithoutJD())
	if err != nil {
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	ae.decoder = decoder.NewLegacyDecoder(
		decoder.WithEVMABIMappings(ae.evmABIMappings),
		decoder.WithSolanaDecoders(ae.solanaDecoders),
	)

	// Decode proposal
	decodedProposal, err := ae.decoder.Decode(ctx, env, proposal)
	if err != nil {
		return nil, fmt.Errorf("failed to decode timelock proposal: %w", err)
	}

	actx := &analyzerContext{}
	ectx := executionContext{
		domain:          domain,
		environmentName: environmentName,
		blockChains:     env.BlockChains,
		dataStore:       env.DataStore,
	}

	ae.executionContext = &ectx

	analyzedProposal, err := ae.analyzeProposal(ctx, actx, ectx, decodedProposal)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze timelock proposal: %w", err)
	}

	return analyzedProposal, nil
}

// Render writes the rendered proposal output to the provided io.Writer.
func (ae *analyzerEngine) Render(
	ctx context.Context,
	w io.Writer,
	rendererID string,
	proposal types.AnalyzedProposal,
) error {
	r, exists := ae.rendererRegistry.Get(rendererID)
	if !exists {
		return fmt.Errorf("renderer %s not registered", rendererID)
	}

	if ae.executionContext == nil {
		return fmt.Errorf("execution context not available - ensure Run() was called before Render()")
	}

	req := types.RendererRequest{
		Domain:          ae.executionContext.Domain().String(),
		EnvironmentName: ae.executionContext.EnvironmentName(),
	}

	return r.Render(ctx, w, req, proposal)
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

func (ae *analyzerEngine) RegisterRenderer(r types.Renderer) error {
	return ae.rendererRegistry.Register(r)
}

func (ae *analyzerEngine) RegisterEVMABIMappings(evmABIMappings map[string]string) error {
	if len(evmABIMappings) == 0 {
		return fmt.Errorf("evm ABI mappings cannot be empty")
	}

	for key, abi := range evmABIMappings {
		if key == "" {
			return fmt.Errorf("evm ABI mapping key cannot be empty")
		}
		if abi == "" {
			return fmt.Errorf("evm ABI mapping value cannot be empty for key %q", key)
		}
		if _, exists := ae.evmABIMappings[key]; exists {
			return fmt.Errorf("evm ABI mapping for key %q is already registered", key)
		}
		ae.evmABIMappings[key] = abi
	}

	return nil
}

func (ae *analyzerEngine) RegisterSolanaDecoders(solanaDecoders map[string]experimentalanalyzer.DecodeInstructionFn) error {
	if len(solanaDecoders) == 0 {
		return fmt.Errorf("solana decoders cannot be empty")
	}

	for key, decodeFn := range solanaDecoders {
		if key == "" {
			return fmt.Errorf("solana decoder key cannot be empty")
		}
		if decodeFn == nil {
			return fmt.Errorf("solana decoder cannot be nil for key %q", key)
		}
		if _, exists := ae.solanaDecoders[key]; exists {
			return fmt.Errorf("solana decoder for key %q is already registered", key)
		}
		ae.solanaDecoders[key] = decodeFn
	}

	return nil
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

type analyzerExecutionResult struct {
	analyzerID  string
	annotations types.Annotations
	err         error
	timedOut    bool
	skipped     bool
}

func executeAnalyzerLevels(
	ctx context.Context,
	levels [][]types.BaseAnalyzer,
	execute func(context.Context, types.BaseAnalyzer) analyzerExecutionResult,
) []analyzerExecutionResult {
	results := make([]analyzerExecutionResult, 0)
	for _, level := range levels {
		levelResults := make([]analyzerExecutionResult, len(level))
		var wg sync.WaitGroup
		for i, baseAnalyzer := range level {
			wg.Add(1)
			go func() {
				defer wg.Done()
				levelResults[i] = execute(ctx, baseAnalyzer)
			}()
		}
		wg.Wait()
		results = append(results, levelResults...)
	}
	return results
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
			ae.logger.Errorw("Failed to analyze batch operation", "chainSelector", batchOp.ChainSelector(), "error", err)
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

	levels := graph.Levels()
	results := executeAnalyzerLevels(ctx, levels, func(ctx context.Context, baseAnalyzer types.BaseAnalyzer) analyzerExecutionResult {
		proposalAnalyzer := baseAnalyzer.(types.ProposalAnalyzer)
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}
		if !proposalAnalyzer.CanAnalyze(ctx, req, decodedProposal) {
			return analyzerExecutionResult{
				analyzerID: proposalAnalyzer.ID(),
				skipped:    true,
			}
		}
		analyzerCtx, cancel := context.WithTimeout(ctx, ae.analyzerTimeout)
		annotations, err := proposalAnalyzer.Analyze(analyzerCtx, req, decodedProposal)
		timedOut := errors.Is(analyzerCtx.Err(), context.DeadlineExceeded)
		cancel() // Always cancel to free resources
		return analyzerExecutionResult{
			analyzerID:  proposalAnalyzer.ID(),
			annotations: annotations,
			err:         err,
			timedOut:    timedOut,
		}
	})
	for _, result := range results {
		if result.skipped {
			continue
		}
		if result.err != nil {
			if result.timedOut {
				ae.logger.Errorw("Proposal analyzer timed out", "analyzerID", result.analyzerID, "timeout", ae.analyzerTimeout)
			} else {
				ae.logger.Errorw("Proposal analyzer failed", "analyzerID", result.analyzerID, "error", result.err)
			}
			continue
		}
		trackedAnnotations := trackAnnotations(result.annotations, result.analyzerID)
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
			ae.logger.Errorw("Failed to analyze call", "callName", call.Name(), "error", err)
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

	levels := graph.Levels()
	results := executeAnalyzerLevels(ctx, levels, func(ctx context.Context, baseAnalyzer types.BaseAnalyzer) analyzerExecutionResult {
		batchOpAnalyzer := baseAnalyzer.(types.BatchOperationAnalyzer)
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}
		if !batchOpAnalyzer.CanAnalyze(ctx, req, decodedBatchOperation) {
			return analyzerExecutionResult{
				analyzerID: batchOpAnalyzer.ID(),
				skipped:    true,
			}
		}
		analyzerCtx, cancel := context.WithTimeout(ctx, ae.analyzerTimeout)
		annotations, err := batchOpAnalyzer.Analyze(analyzerCtx, req, decodedBatchOperation)
		timedOut := analyzerCtx.Err() == context.DeadlineExceeded
		cancel() // Always cancel to free resources
		return analyzerExecutionResult{
			analyzerID:  batchOpAnalyzer.ID(),
			annotations: annotations,
			err:         err,
			timedOut:    timedOut,
		}
	})
	for _, result := range results {
		if result.skipped {
			continue
		}
		if result.err != nil {
			if result.timedOut {
				ae.logger.Errorw("Batch operation analyzer timed out", "analyzerID", result.analyzerID, "chainSelector", decodedBatchOperation.ChainSelector(), "timeout", ae.analyzerTimeout)
			} else {
				ae.logger.Errorw("Batch operation analyzer failed", "analyzerID", result.analyzerID, "chainSelector", decodedBatchOperation.ChainSelector(), "error", result.err)
			}
			continue
		}
		trackedAnnotations := trackAnnotations(result.annotations, result.analyzerID)
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
			ae.logger.Errorw("Failed to analyze input parameter", "paramName", param.Name(), "paramType", param.Type(), "error", err)
			continue
		}
		inputs = append(inputs, analyzedParam)
	}

	outputs := make(types.AnalyzedParameters, 0)
	for _, param := range decodedCall.Outputs() {
		analyzedParam, err := ae.analyzeParameter(ctx, actx, ectx, param)
		if err != nil {
			ae.logger.Errorw("Failed to analyze output parameter", "paramName", param.Name(), "paramType", param.Type(), "error", err)
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

	levels := graph.Levels()
	results := executeAnalyzerLevels(ctx, levels, func(ctx context.Context, baseAnalyzer types.BaseAnalyzer) analyzerExecutionResult {
		callAnalyzer := baseAnalyzer.(types.CallAnalyzer)
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}
		if !callAnalyzer.CanAnalyze(ctx, req, decodedCall) {
			return analyzerExecutionResult{
				analyzerID: callAnalyzer.ID(),
				skipped:    true,
			}
		}
		analyzerCtx, cancel := context.WithTimeout(ctx, ae.analyzerTimeout)
		annotations, err := callAnalyzer.Analyze(analyzerCtx, req, decodedCall)
		timedOut := analyzerCtx.Err() == context.DeadlineExceeded
		cancel() // Always cancel to free resources
		return analyzerExecutionResult{
			analyzerID:  callAnalyzer.ID(),
			annotations: annotations,
			err:         err,
			timedOut:    timedOut,
		}
	})
	for _, result := range results {
		if result.skipped {
			continue
		}
		if result.err != nil {
			if result.timedOut {
				ae.logger.Errorw("Call analyzer timed out", "analyzerID", result.analyzerID, "callName", decodedCall.Name(), "timeout", ae.analyzerTimeout)
			} else {
				ae.logger.Errorw("Call analyzer failed", "analyzerID", result.analyzerID, "callName", decodedCall.Name(), "error", result.err)
			}
			continue
		}
		trackedAnnotations := trackAnnotations(result.annotations, result.analyzerID)
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

	levels := graph.Levels()
	results := executeAnalyzerLevels(ctx, levels, func(ctx context.Context, baseAnalyzer types.BaseAnalyzer) analyzerExecutionResult {
		paramAnalyzer := baseAnalyzer.(types.ParameterAnalyzer)
		req := types.AnalyzerRequest{
			AnalyzerContext:  actx,
			ExecutionContext: ectx,
		}
		if !paramAnalyzer.CanAnalyze(ctx, req, decodedParameter) {
			return analyzerExecutionResult{
				analyzerID: paramAnalyzer.ID(),
				skipped:    true,
			}
		}
		analyzerCtx, cancel := context.WithTimeout(ctx, ae.analyzerTimeout)
		annotations, err := paramAnalyzer.Analyze(analyzerCtx, req, decodedParameter)
		timedOut := analyzerCtx.Err() == context.DeadlineExceeded
		cancel() // Always cancel to free resources
		return analyzerExecutionResult{
			analyzerID:  paramAnalyzer.ID(),
			annotations: annotations,
			err:         err,
			timedOut:    timedOut,
		}
	})
	for _, result := range results {
		if result.skipped {
			continue
		}
		if result.err != nil {
			if result.timedOut {
				ae.logger.Errorw("Parameter analyzer timed out", "analyzerID", result.analyzerID, "paramName", decodedParameter.Name(), "paramType", decodedParameter.Type(), "timeout", ae.analyzerTimeout)
			} else {
				ae.logger.Errorw("Parameter analyzer failed", "analyzerID", result.analyzerID, "paramName", decodedParameter.Name(), "paramType", decodedParameter.Type(), "error", result.err)
			}
			continue
		}
		trackedAnnotations := trackAnnotations(result.annotations, result.analyzerID)
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

func (a analyzedCall) ContractType() string {
	return a.decodedCall.ContractType()
}

func (a analyzedCall) ContractVersion() string {
	return a.decodedCall.ContractVersion()
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
