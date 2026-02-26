package proposalanalysis

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/smartcontractkit/mcms"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	analyzermocks "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/mocks"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

func TestAnalyzerEngineRunInputValidation(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine()
	validReq := RunRequest{
		Domain:      cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{Name: "staging"},
	}

	t.Run("nil proposal", func(t *testing.T) {
		t.Parallel()

		_, err := engine.Run(t.Context(), validReq, nil)
		require.EqualError(t, err, "proposal cannot be nil")
	})

	t.Run("empty domain", func(t *testing.T) {
		t.Parallel()

		_, err := engine.Run(t.Context(), RunRequest{
			Domain:      cldfdomain.Domain{},
			Environment: validReq.Environment,
		}, &mcms.TimelockProposal{})
		require.EqualError(t, err, "domain cannot be empty")
	})

	t.Run("nil environment", func(t *testing.T) {
		t.Parallel()

		_, err := engine.Run(t.Context(), RunRequest{
			Domain:      validReq.Domain,
			Environment: nil,
		}, &mcms.TimelockProposal{})
		require.EqualError(t, err, "environment cannot be nil")
	})
}

func TestAnalyzerEngineRunDecoderFactoryError(t *testing.T) {
	t.Parallel()

	boom := errors.New("factory failed")
	engine := newAnalyzerEngineWithDeps(Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return nil, boom
		},
	})

	_, err := engine.Run(t.Context(), RunRequest{
		Domain:      cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{Name: "staging"},
	}, &mcms.TimelockProposal{})

	require.Error(t, err)
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "build proposal decoder")
}

func TestAnalyzerEngineRunDecodeError(t *testing.T) {
	t.Parallel()

	boom := errors.New("decode failed")
	engine := newAnalyzerEngineWithDeps(Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return decoderStub{
				decodeFn: func(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error) {
					return nil, boom
				},
			}, nil
		},
	})

	_, err := engine.Run(t.Context(), RunRequest{
		Domain:      cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{Name: "staging"},
	}, &mcms.TimelockProposal{})

	require.Error(t, err)
	require.ErrorIs(t, err, boom)
	require.ErrorContains(t, err, "decode proposal")
}

func TestAnalyzerEngineRegisterAnalyzer(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine()

	err := engine.RegisterAnalyzer(nil)
	require.EqualError(t, err, "analyzer cannot be nil")

	a := analyzermocks.NewMockProposalAnalyzer(t)
	a.EXPECT().ID().Return("proposal-a")

	require.NoError(t, engine.RegisterAnalyzer(a))

	err = engine.RegisterAnalyzer(a)
	require.EqualError(t, err, "analyzer with ID \"proposal-a\" is already registered")
}

func TestAnalyzerEngineRegisterRenderer(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine()

	err := engine.RegisterRenderer(nil)
	require.EqualError(t, err, "renderer cannot be nil")

	r := &rendererStub{id: "plain"}
	require.NoError(t, engine.RegisterRenderer(r))

	err = engine.RegisterRenderer(r)
	require.EqualError(t, err, "renderer with ID \"plain\" is already registered")
}

func TestAnalyzerEngineRenderTo(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine()
	out := &bytes.Buffer{}
	req := renderer.RenderRequest{
		Domain:          "mcms",
		EnvironmentName: "staging",
	}

	err := engine.RenderTo(out, "missing", req, analyzer.NewAnalyzedProposalNode(nil))
	require.EqualError(t, err, "renderer with ID \"missing\" is not registered")

	r := &rendererStub{
		id: "plain",
		renderFn: func(w io.Writer, req renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error {
			_, writeErr := w.Write([]byte("rendered"))
			return writeErr
		},
	}
	require.NoError(t, engine.RegisterRenderer(r))

	err = engine.RenderTo(out, "plain", req, analyzer.NewAnalyzedProposalNode(nil))
	require.NoError(t, err)
	require.Equal(t, "rendered", out.String())
}

func TestAnalyzerEngineRunExecutesAnalyzersAndResolvesDependencies(t *testing.T) {
	t.Parallel()

	decoded, batch, call, inputParam, outputParam := newDecodedFixture()
	require.NotNil(t, batch)
	require.NotNil(t, call)
	require.NotNil(t, inputParam)
	require.NotNil(t, outputParam)

	engine := newAnalyzerEngineWithDeps(Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return decoderStub{
				decodeFn: func(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error) {
					return decoded, nil
				},
			}, nil
		},
	})

	a := analyzermocks.NewMockProposalAnalyzer(t)
	a.EXPECT().ID().Return("analyzer-a")
	a.EXPECT().Dependencies().Return(nil)
	a.EXPECT().CanAnalyze(mock.Anything, mock.Anything, decoded).Return(true)
	a.EXPECT().
		Analyze(mock.Anything, mock.Anything, decoded).
		RunAndReturn(func(ctx context.Context, req analyzer.ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
			return annotation.Annotations{annotation.New("dep-annotation", "string", "ok")}, nil
		})

	b := analyzermocks.NewMockProposalAnalyzer(t)
	b.EXPECT().ID().Return("analyzer-b")
	b.EXPECT().Dependencies().Return([]string{"analyzer-a"})
	b.EXPECT().
		CanAnalyze(mock.Anything, mock.Anything, decoded).
		RunAndReturn(func(ctx context.Context, req analyzer.ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) bool {
			depAnns := req.DependencyAnnotationStore.DependencyAnnotations()
			require.Len(t, depAnns, 1)
			require.Equal(t, "dep-annotation", depAnns[0].Name())
			require.Equal(t, "analyzer-a", depAnns[0].AnalyzerID())

			return true
		})
	b.EXPECT().
		Analyze(mock.Anything, mock.Anything, decoded).
		RunAndReturn(func(ctx context.Context, req analyzer.ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
			return annotation.Annotations{annotation.New("final-annotation", "string", "ok")}, nil
		})

	require.NoError(t, engine.RegisterAnalyzer(a))
	require.NoError(t, engine.RegisterAnalyzer(b))

	analyzed, err := engine.Run(t.Context(), RunRequest{
		Domain: cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{
			Name:        "staging",
			DataStore:   datastore.NewMemoryDataStore().Seal(),
			BlockChains: chain.NewBlockChains(nil),
		},
	}, &mcms.TimelockProposal{})
	require.NoError(t, err)
	require.NotNil(t, analyzed)

	anns := analyzed.Annotations()
	require.Len(t, anns, 2)
	require.Equal(t, "dep-annotation", anns[0].Name())
	require.Equal(t, "analyzer-a", anns[0].AnalyzerID())
	require.Equal(t, "final-annotation", anns[1].Name())
	require.Equal(t, "analyzer-b", anns[1].AnalyzerID())
}

type decoderStub struct {
	decodeFn func(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error)
}

func (s decoderStub) Decode(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error) {
	return s.decodeFn(ctx, env, proposal)
}

type rendererStub struct {
	id       string
	renderFn func(w io.Writer, req renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error
}

func (r *rendererStub) ID() string {
	return r.id
}

func (r *rendererStub) RenderTo(w io.Writer, req renderer.RenderRequest, proposal analyzer.AnalyzedProposal) error {
	if r.renderFn == nil {
		return nil
	}

	return r.renderFn(w, req, proposal)
}
