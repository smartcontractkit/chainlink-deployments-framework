package proposalanalysis

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	internalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	internaldecoder "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/decoder"
	internalrenderer "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/renderer"
	internalscheduler "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/scheduler"
)

// RunRequest encapsulates the domain, environment, and decoder configuration
type RunRequest struct {
	Domain        cldfdomain.Domain
	Environment   *deployment.Environment
	DecoderConfig DecoderConfig
}

type AnalyzerEngine interface {
	Run(ctx context.Context, req RunRequest, proposal *mcms.TimelockProposal) (AnalyzedProposal, error)

	RegisterAnalyzer(analyzer BaseAnalyzer) error

	RegisterRenderer(renderer Renderer) error

	RenderTo(w io.Writer, rendererID string, renderReq RenderRequest, proposal AnalyzedProposal) error
}

type deps struct {
	DecoderFactory func(cfg DecoderConfig) (internaldecoder.ProposalDecoder, error)
}

func defaultDeps() deps {
	return deps{
		DecoderFactory: func(cfg DecoderConfig) (internaldecoder.ProposalDecoder, error) {
			return internaldecoder.NewExperimentalDecoder(cfg), nil
		},
	}
}

type analyzerEngine struct {
	analyzerRegistry *internalanalyzer.Registry
	rendererRegistry *internalrenderer.Registry
	config           *engineConfig
	deps             deps
}

// NewAnalyzerEngine creates a new AnalyzerEngine with the given options.
func NewAnalyzerEngine(opts ...EngineOption) AnalyzerEngine {
	return newAnalyzerEngineWithDeps(deps{}, opts...)
}

func newAnalyzerEngineWithDeps(deps deps, opts ...EngineOption) AnalyzerEngine {
	cfg := ApplyEngineOptions(opts...)
	resolvedDeps := defaultDeps()
	if deps.DecoderFactory != nil {
		resolvedDeps.DecoderFactory = deps.DecoderFactory
	}

	return &analyzerEngine{
		analyzerRegistry: internalanalyzer.NewRegistry(),
		rendererRegistry: internalrenderer.NewRegistry(),
		config:           cfg,
		deps:             resolvedDeps,
	}
}

// RegisterAnalyzer registers a new analyzer with the engine.
func (e *analyzerEngine) RegisterAnalyzer(a BaseAnalyzer) error {
	return e.analyzerRegistry.Register(a)
}

// RegisterRenderer registers a new renderer with the engine.
func (e *analyzerEngine) RegisterRenderer(r Renderer) error {
	return e.rendererRegistry.Register(r)
}

// Run analyzes the given proposal and returns the analyzed proposal.
func (e *analyzerEngine) Run(ctx context.Context, req RunRequest, proposal *mcms.TimelockProposal) (AnalyzedProposal, error) {
	if proposal == nil {
		return nil, errors.New("proposal cannot be nil")
	}
	if req.Domain.Key() == "" {
		return nil, errors.New("domain cannot be empty")
	}
	if req.Environment == nil {
		return nil, errors.New("environment cannot be nil")
	}

	proposalDecoder, err := e.deps.DecoderFactory(req.DecoderConfig)
	if err != nil {
		return nil, fmt.Errorf("build proposal decoder: %w", err)
	}

	decodedProposal, err := proposalDecoder.Decode(ctx, *req.Environment, proposal)
	if err != nil {
		return nil, fmt.Errorf("decode proposal: %w", err)
	}

	state := newRunState(req, decodedProposal)
	g, err := internalscheduler.New(e.analyzerRegistry.All())
	if err != nil {
		return state.proposal, fmt.Errorf("build analyzer graph: %w", err)
	}

	run := func(runCtx context.Context, baseAnalyzer internalanalyzer.BaseAnalyzer) error {
		switch a := baseAnalyzer.(type) {
		case internalanalyzer.ProposalAnalyzer:
			return state.runProposalAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case internalanalyzer.BatchOperationAnalyzer:
			return state.runBatchAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case internalanalyzer.CallAnalyzer:
			return state.runCallAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case internalanalyzer.ParameterAnalyzer:
			return state.runParameterAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		default:
			return fmt.Errorf("no analyzer runner matched analyzer %q", baseAnalyzer.ID())
		}
	}

	if err := g.Run(ctx, run); err != nil {
		return state.proposal, err
	}

	return state.proposal, nil
}

// RenderTo renders the given proposal to the given writer using the given renderer.
func (e *analyzerEngine) RenderTo(w io.Writer, rendererID string, renderReq RenderRequest, proposal AnalyzedProposal) error {
	r, ok := e.rendererRegistry.Get(rendererID)
	if !ok {
		return fmt.Errorf("renderer with ID %q is not registered", rendererID)
	}

	return r.RenderTo(w, renderReq, proposal)
}
