package proposalanalysis

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotated"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotationstore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

// runState owns decoded and analyzed proposals and synchronizes annotation writes.
type runState struct {
	mu sync.RWMutex

	req             RunRequest
	decodedProposal decoder.DecodedTimelockProposal
	proposal        *analyzer.AnalyzedProposalNode
}

func newRunState(req RunRequest, decoded decoder.DecodedTimelockProposal) *runState {
	analyzed := toAnalyzedProposal(decoded)
	return &runState{
		req:             req,
		decodedProposal: decoded,
		proposal:        analyzed,
	}
}

func (s *runState) runProposalAnalyzer(ctx context.Context, a analyzer.ProposalAnalyzer, timeout time.Duration) error {
	req := analyzer.ProposalAnalyzeRequest{
		ExecutionContext:          s.executionContext(),
		DependencyAnnotationStore: annotationstore.NewScopedDependencyAnnotationStore(a.Dependencies(), s.proposalAnnotationsByLevel()),
	}

	anns, skipped, err := runWithTimeout(
		ctx,
		timeout,
		func(callCtx context.Context) bool {
			return a.CanAnalyze(callCtx, req, s.decodedProposal)
		},
		func(callCtx context.Context) (annotation.Annotations, error) {
			return a.Analyze(callCtx, req, s.decodedProposal)
		},
	)
	if err != nil {
		return err
	}
	if skipped {
		return nil
	}

	s.addAnnotations(&s.proposal.BaseAnnotated, a.ID(), anns)

	return nil
}

func (s *runState) runBatchAnalyzer(ctx context.Context, a analyzer.BatchOperationAnalyzer, timeout time.Duration) error {
	for batchIdx, batch := range s.decodedProposal.BatchOperations() {
		req := analyzer.AnalyzeRequest[analyzer.BatchOperationAnalyzerContext]{
			AnalyzerContext:           analyzer.NewBatchOperationAnalyzerContextNode(s.decodedProposal),
			ExecutionContext:          s.executionContext(),
			DependencyAnnotationStore: annotationstore.NewScopedDependencyAnnotationStore(a.Dependencies(), s.batchAnnotationsByLevel(batchIdx)),
		}
		anns, skipped, err := runWithTimeout(
			ctx,
			timeout,
			func(callCtx context.Context) bool {
				return a.CanAnalyze(callCtx, req, batch)
			},
			func(callCtx context.Context) (annotation.Annotations, error) {
				return a.Analyze(callCtx, req, batch)
			},
		)
		if err != nil {
			return err
		}
		if skipped {
			continue
		}

		s.addAnnotations(&s.batchAt(batchIdx).BaseAnnotated, a.ID(), anns)
	}

	return nil
}

func (s *runState) runCallAnalyzer(ctx context.Context, a analyzer.CallAnalyzer, timeout time.Duration) error {
	for batchIdx, batch := range s.decodedProposal.BatchOperations() {
		for callIdx, call := range batch.Calls() {
			req := analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext]{
				AnalyzerContext:           analyzer.NewCallAnalyzerContextNode(s.decodedProposal, batch),
				ExecutionContext:          s.executionContext(),
				DependencyAnnotationStore: annotationstore.NewScopedDependencyAnnotationStore(a.Dependencies(), s.callAnnotationsByLevel(batchIdx, callIdx)),
			}
			anns, skipped, err := runWithTimeout(
				ctx,
				timeout,
				func(callCtx context.Context) bool {
					return a.CanAnalyze(callCtx, req, call)
				},
				func(callCtx context.Context) (annotation.Annotations, error) {
					return a.Analyze(callCtx, req, call)
				},
			)
			if err != nil {
				return err
			}
			if skipped {
				continue
			}
			s.addAnnotations(&s.callAt(batchIdx, callIdx).BaseAnnotated, a.ID(), anns)
		}
	}

	return nil
}

func (s *runState) runParameterAnalyzer(ctx context.Context, a analyzer.ParameterAnalyzer, timeout time.Duration) error {
	for batchIdx, batch := range s.decodedProposal.BatchOperations() {
		for callIdx, call := range batch.Calls() {
			if err := s.runParameterSet(ctx, a, timeout, batchIdx, callIdx, batch, call, call.Inputs(), true); err != nil {
				return err
			}
			if err := s.runParameterSet(ctx, a, timeout, batchIdx, callIdx, batch, call, call.Outputs(), false); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *runState) runParameterSet(
	ctx context.Context,
	a analyzer.ParameterAnalyzer,
	timeout time.Duration,
	batchIdx, callIdx int,
	batch decoder.DecodedBatchOperation,
	call decoder.DecodedCall,
	params decoder.DecodedParameters,
	isInput bool,
) error {
	for paramIdx, param := range params {
		req := analyzer.AnalyzeRequest[analyzer.ParameterAnalyzerContext]{
			AnalyzerContext:           analyzer.NewParameterAnalyzerContextNode(s.decodedProposal, batch, call),
			ExecutionContext:          s.executionContext(),
			DependencyAnnotationStore: annotationstore.NewScopedDependencyAnnotationStore(a.Dependencies(), s.parameterAnnotationsByLevel(batchIdx, callIdx, isInput, paramIdx)),
		}

		anns, skipped, err := runWithTimeout(
			ctx,
			timeout,
			func(callCtx context.Context) bool {
				return a.CanAnalyze(callCtx, req, param)
			},
			func(callCtx context.Context) (annotation.Annotations, error) {
				return a.Analyze(callCtx, req, param)
			},
		)
		if err != nil {
			return err
		}
		if skipped {
			continue
		}

		target := s.outputParameterAt(batchIdx, callIdx, paramIdx)
		if isInput {
			target = s.inputParameterAt(batchIdx, callIdx, paramIdx)
		}
		s.addAnnotations(&target.BaseAnnotated, a.ID(), anns)
	}

	return nil
}

func (s *runState) executionContext() analyzer.ExecutionContext {
	return analyzer.NewExecutionContextNode(
		s.req.Domain,
		s.req.Environment.Name,
		s.req.Environment.BlockChains,
		s.req.Environment.DataStore,
	)
}

func (s *runState) batchAt(batchIdx int) *analyzer.AnalyzedBatchOperationNode {
	return s.proposal.BatchOperations()[batchIdx].(*analyzer.AnalyzedBatchOperationNode)
}

func (s *runState) callAt(batchIdx, callIdx int) *analyzer.AnalyzedCallNode {
	return s.batchAt(batchIdx).Calls()[callIdx].(*analyzer.AnalyzedCallNode)
}

func (s *runState) inputParameterAt(batchIdx, callIdx, paramIdx int) *analyzer.AnalyzedParameterNode {
	return s.callAt(batchIdx, callIdx).Inputs()[paramIdx].(*analyzer.AnalyzedParameterNode)
}

func (s *runState) outputParameterAt(batchIdx, callIdx, paramIdx int) *analyzer.AnalyzedParameterNode {
	return s.callAt(batchIdx, callIdx).Outputs()[paramIdx].(*analyzer.AnalyzedParameterNode)
}

func (s *runState) proposalAnnotationsByLevel() map[annotationstore.AnnotationLevel]annotation.Annotations {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[annotationstore.AnnotationLevel]annotation.Annotations{
		annotationstore.AnnotationLevelProposal: cloneAnnotations(s.proposal.Annotations()),
	}
}

func (s *runState) batchAnnotationsByLevel(batchIdx int) map[annotationstore.AnnotationLevel]annotation.Annotations {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[annotationstore.AnnotationLevel]annotation.Annotations{
		annotationstore.AnnotationLevelProposal:       cloneAnnotations(s.proposal.Annotations()),
		annotationstore.AnnotationLevelBatchOperation: cloneAnnotations(s.batchAt(batchIdx).Annotations()),
	}
}

func (s *runState) callAnnotationsByLevel(batchIdx, callIdx int) map[annotationstore.AnnotationLevel]annotation.Annotations {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[annotationstore.AnnotationLevel]annotation.Annotations{
		annotationstore.AnnotationLevelProposal:       cloneAnnotations(s.proposal.Annotations()),
		annotationstore.AnnotationLevelBatchOperation: cloneAnnotations(s.batchAt(batchIdx).Annotations()),
		annotationstore.AnnotationLevelCall:           cloneAnnotations(s.callAt(batchIdx, callIdx).Annotations()),
	}
}

func (s *runState) parameterAnnotationsByLevel(
	batchIdx, callIdx int,
	isInput bool,
	paramIdx int,
) map[annotationstore.AnnotationLevel]annotation.Annotations {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paramNode := s.outputParameterAt(batchIdx, callIdx, paramIdx)
	if isInput {
		paramNode = s.inputParameterAt(batchIdx, callIdx, paramIdx)
	}

	return map[annotationstore.AnnotationLevel]annotation.Annotations{
		annotationstore.AnnotationLevelProposal:       cloneAnnotations(s.proposal.Annotations()),
		annotationstore.AnnotationLevelBatchOperation: cloneAnnotations(s.batchAt(batchIdx).Annotations()),
		annotationstore.AnnotationLevelCall:           cloneAnnotations(s.callAt(batchIdx, callIdx).Annotations()),
		annotationstore.AnnotationLevelParameter:      cloneAnnotations(paramNode.Annotations()),
	}
}

func cloneAnnotations(anns annotation.Annotations) annotation.Annotations {
	if len(anns) == 0 {
		return nil
	}
	out := make(annotation.Annotations, len(anns))
	copy(out, anns)

	return out
}

func (s *runState) addAnnotations(target *annotated.BaseAnnotated, analyzerID string, anns annotation.Annotations) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tagged := make([]annotation.Annotation, 0, len(anns))
	for _, ann := range anns {
		if ann == nil {
			continue
		}
		// associate the annotation with the analyzer that produced it
		tagged = append(tagged, annotation.NewWithAnalyzer(ann.Name(), ann.Type(), ann.Value(), analyzerID))
	}
	target.AddAnnotations(tagged...)
}

func toAnalyzedProposal(decoded decoder.DecodedTimelockProposal) *analyzer.AnalyzedProposalNode {
	batches := make(analyzer.AnalyzedBatchOperations, 0, len(decoded.BatchOperations()))
	for _, b := range decoded.BatchOperations() {
		calls := make(analyzer.AnalyzedCalls, 0, len(b.Calls()))
		for _, c := range b.Calls() {
			inputs := make(analyzer.AnalyzedParameters, 0, len(c.Inputs()))
			for _, p := range c.Inputs() {
				inputs = append(inputs, analyzer.NewAnalyzedParameterNode(p.Name(), p.Type(), p.Value()))
			}
			outputs := make(analyzer.AnalyzedParameters, 0, len(c.Outputs()))
			for _, p := range c.Outputs() {
				outputs = append(outputs, analyzer.NewAnalyzedParameterNode(p.Name(), p.Type(), p.Value()))
			}
			calls = append(calls, analyzer.NewAnalyzedCallNode(
				c.To(),
				c.Name(),
				inputs,
				outputs,
				c.Data(),
				c.ContractType(),
				c.ContractVersion(),
				parseAdditionalFields(c.AdditionalFields()),
			))
		}
		batches = append(batches, analyzer.NewAnalyzedBatchOperationNode(b.ChainSelector(), calls))
	}

	return analyzer.NewAnalyzedProposalNode(batches)
}

func parseAdditionalFields(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil
	}

	return fields
}
