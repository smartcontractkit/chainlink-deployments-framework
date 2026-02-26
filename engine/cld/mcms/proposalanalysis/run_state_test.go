package proposalanalysis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotationstore"
	analyzermocks "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/mocks"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

func TestRunStateRunProposalAnalyzer_UsesExecutionContextAndDependencyStore(t *testing.T) {
	t.Parallel()

	decoded, batch, call, inputParam, outputParam := newDecodedFixture()
	require.NotNil(t, batch)
	require.NotNil(t, call)
	require.NotNil(t, inputParam)
	require.NotNil(t, outputParam)

	state := newTestRunState(decoded)
	state.proposal.AddAnnotations(
		annotation.NewWithAnalyzer("dep", "string", "allowed", "dep-a"),
		annotation.NewWithAnalyzer("nondep", "string", "blocked", "other"),
	)

	a := analyzermocks.NewMockProposalAnalyzer(t)
	a.EXPECT().ID().Return("proposal-analyzer")
	a.EXPECT().Dependencies().Return([]string{"dep-a"})
	a.EXPECT().
		CanAnalyze(mock.Anything, mock.Anything, decoded).
		RunAndReturn(func(ctx context.Context, req analyzer.ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) bool {
			require.Equal(t, "staging", req.ExecutionContext.EnvironmentName())
			require.Equal(t, "mcms", req.ExecutionContext.Domain().Key())
			require.Equal(t, chain.NewBlockChains(nil), req.ExecutionContext.BlockChains())
			require.NotNil(t, req.ExecutionContext.DataStore())

			depAnns := req.DependencyAnnotationStore.DependencyAnnotations()
			require.Len(t, depAnns, 1)
			require.Equal(t, "dep", depAnns[0].Name())

			return true
		})
	a.EXPECT().
		Analyze(mock.Anything, mock.Anything, decoded).
		RunAndReturn(func(ctx context.Context, req analyzer.ProposalAnalyzeRequest, proposal decoder.DecodedTimelockProposal) (annotation.Annotations, error) {
			return annotation.Annotations{annotation.New("risk", "enum", "high")}, nil
		})

	err := state.runProposalAnalyzer(t.Context(), a, 2*time.Second)
	require.NoError(t, err)

	all := state.proposal.Annotations()
	require.Len(t, all, 3)
	require.Equal(t, "proposal-analyzer", all[2].AnalyzerID())
	require.Equal(t, "risk", all[2].Name())
}

func TestRunStateRunCallAnalyzer_UsesCallContextAndLevelScopedStore(t *testing.T) {
	t.Parallel()

	decoded, batch, call, _, _ := newDecodedFixture()
	state := newTestRunState(decoded)
	state.proposal.AddAnnotations(
		annotation.NewWithAnalyzer("proposal-dep", "string", "ok", "dep-a"),
		annotation.NewWithAnalyzer("proposal-other", "string", "skip", "other"),
	)
	state.batchAt(0).AddAnnotations(
		annotation.NewWithAnalyzer("batch-dep", "string", "ok", "dep-a"),
		annotation.NewWithAnalyzer("batch-other", "string", "skip", "other"),
	)
	state.callAt(0, 0).AddAnnotations(
		annotation.NewWithAnalyzer("call-dep", "string", "ok", "dep-a"),
		annotation.NewWithAnalyzer("call-other", "string", "skip", "other"),
	)

	a := analyzermocks.NewMockCallAnalyzer(t)
	a.EXPECT().ID().Return("call-analyzer")
	a.EXPECT().Dependencies().Return([]string{"dep-a"})
	a.EXPECT().
		CanAnalyze(mock.Anything, mock.Anything, call).
		RunAndReturn(func(ctx context.Context, req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext], got decoder.DecodedCall) bool {
			require.Equal(t, batch, req.AnalyzerContext.BatchOperation())
			require.Equal(t, decoded, req.AnalyzerContext.Proposal())
			require.Equal(t, call, got)

			require.Len(t, req.DependencyAnnotationStore.Filter(annotationstore.ByLevel(annotationstore.AnnotationLevelProposal)), 1)
			require.Len(t, req.DependencyAnnotationStore.Filter(annotationstore.ByLevel(annotationstore.AnnotationLevelBatchOperation)), 1)
			require.Len(t, req.DependencyAnnotationStore.Filter(annotationstore.ByLevel(annotationstore.AnnotationLevelCall)), 1)
			require.Len(t, req.DependencyAnnotationStore.DependencyAnnotations(), 3)

			return true
		})
	a.EXPECT().
		Analyze(mock.Anything, mock.Anything, call).
		RunAndReturn(func(ctx context.Context, req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext], got decoder.DecodedCall) (annotation.Annotations, error) {
			return annotation.Annotations{annotation.New("call-risk", "enum", "medium")}, nil
		})

	err := state.runCallAnalyzer(t.Context(), a, 2*time.Second)
	require.NoError(t, err)

	callAnns := state.callAt(0, 0).Annotations()
	require.Len(t, callAnns, 3)
	require.Equal(t, "call-analyzer", callAnns[2].AnalyzerID())
	require.Equal(t, "call-risk", callAnns[2].Name())
}

func TestRunStateRunParameterAnalyzer_AnnotatesInputAndOutputNodes(t *testing.T) {
	t.Parallel()

	decoded, batch, call, inParam, outParam := newDecodedFixture()
	state := newTestRunState(decoded)
	state.inputParameterAt(0, 0, 0).AddAnnotations(
		annotation.NewWithAnalyzer("input-dep", "string", "ok", "dep-a"),
		annotation.NewWithAnalyzer("input-other", "string", "skip", "other"),
	)
	state.outputParameterAt(0, 0, 0).AddAnnotations(
		annotation.NewWithAnalyzer("output-dep", "string", "ok", "dep-a"),
		annotation.NewWithAnalyzer("output-other", "string", "skip", "other"),
	)

	seen := map[string]int{}
	a := analyzermocks.NewMockParameterAnalyzer(t)
	a.EXPECT().ID().Return("param-analyzer")
	a.EXPECT().Dependencies().Return([]string{"dep-a"})
	a.EXPECT().
		CanAnalyze(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req analyzer.AnalyzeRequest[analyzer.ParameterAnalyzerContext], param decoder.DecodedParameter) bool {
			require.Equal(t, decoded, req.AnalyzerContext.Proposal())
			require.Equal(t, batch, req.AnalyzerContext.BatchOperation())
			require.Equal(t, call, req.AnalyzerContext.Call())

			if param.Name() == inParam.Name() {
				require.Len(t, req.DependencyAnnotationStore.Filter(annotationstore.ByLevel(annotationstore.AnnotationLevelParameter)), 1)
			}
			if param.Name() == outParam.Name() {
				require.Len(t, req.DependencyAnnotationStore.Filter(annotationstore.ByLevel(annotationstore.AnnotationLevelParameter)), 1)
			}

			return true
		})
	a.EXPECT().
		Analyze(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, req analyzer.AnalyzeRequest[analyzer.ParameterAnalyzerContext], param decoder.DecodedParameter) (annotation.Annotations, error) {
			seen[param.Name()]++
			return annotation.Annotations{
				annotation.New("param-note-"+param.Name(), "string", "ok"),
			}, nil
		})

	err := state.runParameterAnalyzer(t.Context(), a, 2*time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, seen[inParam.Name()])
	require.Equal(t, 1, seen[outParam.Name()])

	inAnns := state.inputParameterAt(0, 0, 0).Annotations()
	require.Len(t, inAnns, 3)
	require.Equal(t, "param-analyzer", inAnns[2].AnalyzerID())

	outAnns := state.outputParameterAt(0, 0, 0).Annotations()
	require.Len(t, outAnns, 3)
	require.Equal(t, "param-analyzer", outAnns[2].AnalyzerID())
}

func TestNewRunState_ConvertsAdditionalFieldsFromDecodedCall(t *testing.T) {
	t.Parallel()

	decoded, batch, call, inputParam, outputParam := newDecodedFixture()
	require.NotNil(t, batch)
	require.NotNil(t, call)
	require.NotNil(t, inputParam)
	require.NotNil(t, outputParam)
	state := newTestRunState(decoded)

	require.Equal(t, map[string]any{
		"gas":    float64(12345),
		"strict": true,
		"label":  "router-update",
	}, state.callAt(0, 0).AdditionalFields())
}

func newTestRunState(decoded decoder.DecodedTimelockProposal) *runState {
	req := RunRequest{
		Domain: cldfdomain.NewDomain("/tmp/domains", "mcms"),
		Environment: &deployment.Environment{
			Name:        "staging",
			BlockChains: chain.NewBlockChains(nil),
			DataStore:   datastore.NewMemoryDataStore().Seal(),
		},
	}

	return newRunState(req, decoded)
}

func newDecodedFixture() (
	decoder.DecodedTimelockProposal,
	decoder.DecodedBatchOperation,
	decoder.DecodedCall,
	decoder.DecodedParameter,
	decoder.DecodedParameter,
) {
	in := &decodedParam{name: "amount", atype: "uint256", value: 123}
	out := &decodedParam{name: "ok", atype: "bool", value: true}
	call := &decodedCall{
		to:         "0xabc",
		name:       "transfer",
		inputs:     decoder.DecodedParameters{in},
		outputs:    decoder.DecodedParameters{out},
		additional: json.RawMessage(`{"gas":12345,"strict":true,"label":"router-update"}`),
	}
	batch := &decodedBatch{selector: 111, calls: decoder.DecodedCalls{call}}
	proposal := &decodedProposal{batches: decoder.DecodedBatchOperations{batch}}

	return proposal, batch, call, in, out
}

type decodedProposal struct {
	batches decoder.DecodedBatchOperations
}

func (d *decodedProposal) BatchOperations() decoder.DecodedBatchOperations { return d.batches }

type decodedBatch struct {
	selector uint64
	calls    decoder.DecodedCalls
}

func (d *decodedBatch) ChainSelector() uint64       { return d.selector }
func (d *decodedBatch) Calls() decoder.DecodedCalls { return d.calls }

type decodedCall struct {
	to         string
	name       string
	inputs     decoder.DecodedParameters
	outputs    decoder.DecodedParameters
	additional json.RawMessage
}

func (d *decodedCall) To() string                         { return d.to }
func (d *decodedCall) Name() string                       { return d.name }
func (d *decodedCall) Inputs() decoder.DecodedParameters  { return d.inputs }
func (d *decodedCall) Outputs() decoder.DecodedParameters { return d.outputs }
func (d *decodedCall) Data() []byte                       { return nil }
func (d *decodedCall) AdditionalFields() json.RawMessage  { return d.additional }
func (d *decodedCall) ContractType() string               { return "token" }
func (d *decodedCall) ContractVersion() string            { return "v1" }

type decodedParam struct {
	name  string
	atype string
	value any
}

func (d *decodedParam) Name() string { return d.name }
func (d *decodedParam) Type() string { return d.atype }
func (d *decodedParam) Value() any   { return d.value }
