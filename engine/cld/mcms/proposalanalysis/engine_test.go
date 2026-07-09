package proposalanalysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	mcmsbindings "github.com/smartcontractkit/mcms/sdk/evm/bindings"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfevm "github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/builtin/timelockdelay"
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

func TestNewAnalyzerEngine_registersBuiltinTimelockDelayValidator(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine().(*analyzerEngine)
	ids := make([]string, 0, len(engine.analyzerRegistry.All()))
	for _, a := range engine.analyzerRegistry.All() {
		ids = append(ids, a.ID())
	}

	require.Contains(t, ids, timelockdelay.ValidatorID)
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

func TestAnalyzerEngineRenderTo_clonesTimelockProposal(t *testing.T) {
	t.Parallel()

	original := &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindTimelockProposal,
			ValidUntil:  uint32(time.Now().Add(time.Hour).Unix()), //nolint:gosec // test fixture
			Description: "original",
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): {
					MCMAddress: "0x1111111111111111111111111111111111111111",
				},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector): "0x1111111111111111111111111111111111111111",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
				Transactions:  []mcmstypes.Transaction{{To: "0x123", Data: []byte{0x01}}},
			},
		},
	}
	engine := NewAnalyzerEngine()
	require.NoError(t, engine.RegisterRenderer(&rendererStub{
		id: "mutator",
		renderFn: func(_ io.Writer, req renderer.RenderRequest, _ analyzer.AnalyzedProposal) error {
			req.TimelockProposal.Description = "mutated"
			return nil
		},
	}))

	err := engine.RenderTo(io.Discard, "mutator", renderer.RenderRequest{
		TimelockProposal: original,
	}, analyzer.NewAnalyzedProposalNode(nil))
	require.NoError(t, err)
	require.Equal(t, "original", original.Description)
}

func TestAnalyzerEngineRun_clonesTimelockProposal(t *testing.T) {
	t.Parallel()

	original := validTestTimelockProposal()
	original.Description = "original"

	engine := newAnalyzerEngineWithDeps(Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return decoderStub{
				decodeFn: func(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error) {
					return &decodedProposalStub{}, nil
				},
			}, nil
		},
	})
	require.NoError(t, engine.RegisterAnalyzer(mutatingRunProposalAnalyzer{}))

	_, err := engine.Run(t.Context(), RunRequest{
		Domain:      cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{Name: "staging"},
	}, original)
	require.NoError(t, err)
	require.Equal(t, "original", original.Description)
}

func TestAnalyzerEngineRun_clonesTimelockProposalPerAnalyzerCall(t *testing.T) {
	t.Parallel()

	original := validTestTimelockProposal()
	original.Description = "original"

	engine := newAnalyzerEngineWithDeps(Deps{
		DecoderFactory: func(cfg decoder.Config) (decoder.ProposalDecoder, error) {
			return decoderStub{
				decodeFn: func(ctx context.Context, env deployment.Environment, proposal *mcms.TimelockProposal) (decoder.DecodedTimelockProposal, error) {
					return &decodedProposalStub{}, nil
				},
			}, nil
		},
	})
	require.NoError(t, engine.RegisterAnalyzer(mutatingRunProposalAnalyzer{}))
	require.NoError(t, engine.RegisterAnalyzer(proposalDescriptionReaderAnalyzer{t: t}))

	_, err := engine.Run(t.Context(), RunRequest{
		Domain:      cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{Name: "staging"},
	}, original)
	require.NoError(t, err)
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
	}, bypassTestTimelockProposal())
	require.NoError(t, err)
	require.NotNil(t, analyzed)

	anns := analyzed.Annotations()
	require.Len(t, anns, 2)
	require.Equal(t, "dep-annotation", anns[0].Name())
	require.Equal(t, "analyzer-a", anns[0].AnalyzerID())
	require.Equal(t, "final-annotation", anns[1].Name())
	require.Equal(t, "analyzer-b", anns[1].AnalyzerID())
}

func TestAnalyzerEngineE2E_RunAndRenderMarkdown(t *testing.T) {
	t.Parallel()

	out := renderE2EMarkdown(t)

	goldenPath := filepath.Join("testdata", "golden_e2e_markdown.md")
	if os.Getenv("UPDATE_GOLDEN") == "true" {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, []byte(out), 0o644))
	}

	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "golden file missing — run UPDATE_GOLDEN=true go test ./engine/cld/mcms/proposalanalysis -run TestAnalyzerEngineE2E_RunAndRenderMarkdown")
	require.Equal(t, string(expected), out)
}

func renderE2EMarkdown(t *testing.T) string {
	t.Helper()

	engine := NewAnalyzerEngine()

	require.NoError(t, engine.RegisterAnalyzer(staticProposalNoteAnalyzer{}))

	markdownRenderer, err := renderer.NewMarkdownRenderer()
	require.NoError(t, err)
	require.NoError(t, engine.RegisterRenderer(markdownRenderer))

	proposal := validTestTimelockProposalWithOperations([]mcmstypes.BatchOperation{
		{
			ChainSelector: mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector),
			Transactions: []mcmstypes.Transaction{
				{
					To:               "0x1234567890123456789012345678901234567890",
					Data:             []byte{},
					AdditionalFields: json.RawMessage(`{"value":1000000000000000000}`),
				},
			},
		},
	})

	analyzed, err := engine.Run(t.Context(), RunRequest{
		Domain: cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{
			Name:              "staging",
			ExistingAddresses: deployment.NewMemoryAddressBook(),
			DataStore:         datastore.NewMemoryDataStore().Seal(),
			BlockChains:       e2eTimelockBlockChains(t),
		},
	}, proposal)
	require.NoError(t, err)

	var out bytes.Buffer
	err = engine.RenderTo(&out, renderer.IDMarkdown, renderer.RenderRequest{
		Domain:          "mcms",
		EnvironmentName: "staging",
	}, analyzed)
	require.NoError(t, err)

	return out.String()
}

func e2eTimelockBlockChains(t *testing.T) chain.BlockChains {
	t.Helper()

	selector := chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector
	mockClient := cldfevm.NewMockOnchainClient(t)

	parsedABI, err := mcmsbindings.RBACTimelockMetaData.GetAbi()
	require.NoError(t, err)

	// 30m on-chain minDelay; proposal delay in the fixture is 1h.
	encoded, err := parsedABI.Methods["getMinDelay"].Outputs.Pack(big.NewInt(30 * 60))
	require.NoError(t, err)

	mockClient.EXPECT().
		CallContract(mock.Anything, mock.Anything, mock.Anything).
		Return(encoded, nil).
		Once()

	return chain.NewBlockChains(map[uint64]chain.BlockChain{
		selector: cldfevm.Chain{
			Selector: selector,
			Client:   mockClient,
		},
	})
}

func TestAnalyzerEngineRenderTo_MarkdownPreservesLargeJSONNumber(t *testing.T) {
	t.Parallel()

	engine := NewAnalyzerEngine()

	markdownRenderer, err := renderer.NewMarkdownRenderer()
	require.NoError(t, err)
	require.NoError(t, engine.RegisterRenderer(markdownRenderer))

	const large = "200000000000000000111"
	proposal := analyzer.NewAnalyzedProposalNode(analyzer.AnalyzedBatchOperations{
		analyzer.NewAnalyzedBatchOperationNode(5009297550715157269, analyzer.AnalyzedCalls{
			analyzer.NewAnalyzedCallNode(
				"0x1111111111111111111111111111111111111111",
				"setConfig",
				analyzer.AnalyzedParameters{
					analyzer.NewAnalyzedParameterNode("maxFee", "uint256", json.Number(large)),
				},
				nil, nil, "Router", "v1.0.0", nil,
			),
		}),
	})

	var out bytes.Buffer
	err = engine.RenderTo(&out, renderer.IDMarkdown, renderer.RenderRequest{
		Domain:          "mcms",
		EnvironmentName: "staging",
	}, proposal)
	require.NoError(t, err)

	require.Contains(t, out.String(), "200,000,000,000,000,000,111")
	require.NotContains(t, out.String(), "2e+20")
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

type staticProposalNoteAnalyzer struct{}

func (staticProposalNoteAnalyzer) ID() string {
	return "static-proposal-note"
}

func (staticProposalNoteAnalyzer) Dependencies() []string {
	// depends on the timelock delay validator so golden test remains stable
	return []string{timelockdelay.ValidatorID}
}

func (staticProposalNoteAnalyzer) CanAnalyze(context.Context, analyzer.ProposalAnalyzeRequest, decoder.DecodedTimelockProposal) bool {
	return true
}

func (staticProposalNoteAnalyzer) Analyze(context.Context, analyzer.ProposalAnalyzeRequest, decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
	return annotation.Annotations{
		annotation.New("proposal.note", "string", "generated by analyzer"),
	}, nil
}

type mutatingRunProposalAnalyzer struct{}

func (mutatingRunProposalAnalyzer) ID() string             { return "run-mutator" }
func (mutatingRunProposalAnalyzer) Dependencies() []string { return nil }

func (mutatingRunProposalAnalyzer) CanAnalyze(context.Context, analyzer.ProposalAnalyzeRequest, decoder.DecodedTimelockProposal) bool {
	return true
}

func (mutatingRunProposalAnalyzer) Analyze(_ context.Context, req analyzer.ProposalAnalyzeRequest, _ decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
	if req.TimelockProposal != nil {
		req.TimelockProposal.Description = "mutated"
	}

	return nil, nil
}

type proposalDescriptionReaderAnalyzer struct {
	t *testing.T
}

func (a proposalDescriptionReaderAnalyzer) ID() string           { return "proposal-description-reader" }
func (proposalDescriptionReaderAnalyzer) Dependencies() []string { return nil }

func (proposalDescriptionReaderAnalyzer) CanAnalyze(context.Context, analyzer.ProposalAnalyzeRequest, decoder.DecodedTimelockProposal) bool {
	return true
}

func (a proposalDescriptionReaderAnalyzer) Analyze(_ context.Context, req analyzer.ProposalAnalyzeRequest, _ decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
	require.NotNil(a.t, req.TimelockProposal)
	require.Equal(a.t, "original", req.TimelockProposal.Description)

	return nil, nil
}

type decodedProposalStub struct{}

func (decodedProposalStub) BatchOperations() decoder.DecodedBatchOperations {
	return nil
}

func validTestTimelockProposal() *mcms.TimelockProposal {
	chainSelector := mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA.Selector)

	return &mcms.TimelockProposal{
		BaseProposal: mcms.BaseProposal{
			Version:     "v1",
			Kind:        mcmstypes.KindTimelockProposal,
			ValidUntil:  uint32(time.Now().Add(time.Hour).Unix()), //nolint:gosec // test fixture
			Description: "test proposal",
			ChainMetadata: map[mcmstypes.ChainSelector]mcmstypes.ChainMetadata{
				chainSelector: {MCMAddress: "0x1111111111111111111111111111111111111111"},
			},
		},
		Action: mcmstypes.TimelockActionSchedule,
		Delay:  mcmstypes.NewDuration(time.Hour),
		TimelockAddresses: map[mcmstypes.ChainSelector]string{
			chainSelector: "0x2222222222222222222222222222222222222222",
		},
		Operations: []mcmstypes.BatchOperation{
			{
				ChainSelector: chainSelector,
				Transactions: []mcmstypes.Transaction{
					{To: "0x1111111111111111111111111111111111111111", Data: []byte{0x01}},
				},
			},
		},
	}
}

func validTestTimelockProposalWithOperations(ops []mcmstypes.BatchOperation) *mcms.TimelockProposal {
	proposal := validTestTimelockProposal()
	proposal.Operations = ops

	return proposal
}

func bypassTestTimelockProposal() *mcms.TimelockProposal {
	proposal := validTestTimelockProposal()
	proposal.Action = mcmstypes.TimelockActionBypass

	return proposal
}
