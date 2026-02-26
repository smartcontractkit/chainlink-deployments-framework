package proposalanalysis

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/renderer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/scheduler"
)

// RunRequest encapsulates the domain, environment, and decoder configuration
type RunRequest struct {
	Domain        cldfdomain.Domain
	Environment   *deployment.Environment
	DecoderConfig decoder.Config
}

type AnalyzerEngine interface {
	Run(ctx context.Context, req RunRequest, proposal *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error)

	RegisterAnalyzer(analyzer analyzer.BaseAnalyzer) error

	RegisterRenderer(renderer renderer.Renderer) error

	RenderTo(w io.Writer, rendererID string, renderReq renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error
}

type Deps struct {
	DecoderFactory func(cfg decoder.Config) (decoder.ProposalDecoder, error)
}

func defaultDeps() Deps {
	return Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return decoder.NewExperimentalDecoder(cfg), nil
		},
	}
}

type analyzerEngine struct {
	analyzerRegistry *analyzer.Registry
	rendererRegistry *renderer.Registry
	config           *engineConfig
	deps             Deps
}

// NewAnalyzerEngine creates a new AnalyzerEngine with the given options.
func NewAnalyzerEngine(opts ...EngineOption) AnalyzerEngine {
	return newAnalyzerEngineWithDeps(Deps{}, opts...)
}

func newAnalyzerEngineWithDeps(deps Deps, opts ...EngineOption) AnalyzerEngine {
	cfg := ApplyEngineOptions(opts...)
	resolvedDeps := defaultDeps()
	if deps.DecoderFactory != nil {
		resolvedDeps.DecoderFactory = deps.DecoderFactory
	}

	return &analyzerEngine{
		analyzerRegistry: analyzer.NewRegistry(),
		rendererRegistry: renderer.NewRegistry(),
		config:           cfg,
		deps:             resolvedDeps,
	}
}

// RegisterAnalyzer registers a new analyzer with the engine.
func (e *analyzerEngine) RegisterAnalyzer(a analyzer.BaseAnalyzer) error {
	return e.analyzerRegistry.Register(a)
}

// RegisterRenderer registers a new renderer with the engine.
func (e *analyzerEngine) RegisterRenderer(r renderer.Renderer) error {
	return e.rendererRegistry.Register(r)
}

// Run analyzes the given proposal and returns the analyzed proposal.
func (e *analyzerEngine) Run(ctx context.Context, req RunRequest, proposal *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error) {
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
	g, err := scheduler.New(e.analyzerRegistry.All())
	if err != nil {
		return state.proposal, fmt.Errorf("build analyzer graph: %w", err)
	}

	run := func(runCtx context.Context, baseAnalyzer analyzer.BaseAnalyzer) error {
		switch a := baseAnalyzer.(type) {
		case analyzer.ProposalAnalyzer:
			return state.runProposalAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case analyzer.BatchOperationAnalyzer:
			return state.runBatchAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case analyzer.CallAnalyzer:
			return state.runCallAnalyzer(runCtx, a, e.config.GetAnalyzerTimeout())
		case analyzer.ParameterAnalyzer:
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
func (e *analyzerEngine) RenderTo(w io.Writer, rendererID string, renderReq renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error {
	r, ok := e.rendererRegistry.Get(rendererID)
	if !ok {
		return fmt.Errorf("renderer with ID %q is not registered", rendererID)
	}

	return r.RenderTo(w, renderReq, proposal)
}
