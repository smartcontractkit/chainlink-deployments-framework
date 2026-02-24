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
)

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
		// DecoderFactory: decoder.New,
	}
}

type analyzerEngine struct {
	analyzerRegistry *analyzer.Registry
	rendererRegistry *renderer.Registry
	config           *engineConfig
	deps             Deps
}

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

func (e *analyzerEngine) RegisterAnalyzer(a analyzer.BaseAnalyzer) error {
	return e.analyzerRegistry.Register(a)
}

func (e *analyzerEngine) RegisterRenderer(r renderer.Renderer) error {
	return e.rendererRegistry.Register(r)
}

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

	// todo
	// decode
	// analyze

	return nil, errors.New("not implemented")
}

func (e *analyzerEngine) RenderTo(w io.Writer, rendererID string, renderReq renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error {
	r, ok := e.rendererRegistry.Get(rendererID)
	if !ok {
		return fmt.Errorf("renderer with ID %q is not registered", rendererID)
	}

	return r.RenderTo(w, renderReq, proposal)
}
