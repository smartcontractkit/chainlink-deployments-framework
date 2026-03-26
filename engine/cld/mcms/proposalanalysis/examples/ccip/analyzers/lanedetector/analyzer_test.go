package lanedetector

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

const (
	seiMainnet  uint64 = 9027416829622342829
	avaxMainnet uint64 = 6433500567565415381
	ethMainnet  uint64 = 5009297550715157269
)

func TestAnalyze_SymmetricPairDetectsLane(t *testing.T) {
	t.Parallel()

	proposal := symmetricProposal(seiMainnet, avaxMainnet)

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	require.Len(t, anns, 1)

	val := anns[0].Value().(string)
	assert.Contains(t, val, "<->")
	assert.Contains(t, val, "avalanche-mainnet")
	assert.Contains(t, val, "sei-mainnet")
}

func TestAnalyze_AsymmetricNoLane(t *testing.T) {
	t.Parallel()

	proposal := &stubProposal{batches: []analyzer.DecodedBatchOperation{
		batch(seiMainnet, call("BurnMintTokenPool", "applyChainUpdates", avaxMainnet)),
	}}

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	assert.Empty(t, anns)
}

func TestAnalyze_SelfLoopIgnored(t *testing.T) {
	t.Parallel()

	proposal := &stubProposal{batches: []analyzer.DecodedBatchOperation{
		batch(seiMainnet, call("BurnMintTokenPool", "applyChainUpdates", seiMainnet)),
	}}

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	assert.Empty(t, anns)
}

func TestAnalyze_ThreeChainTopology(t *testing.T) {
	t.Parallel()

	const jovay uint64 = 1523760397290643893
	const abstract uint64 = 3577778157919314504

	proposal := &stubProposal{batches: []analyzer.DecodedBatchOperation{
		batch(jovay, call("BurnMintTokenPool", "applyChainUpdates", ethMainnet)),
		batch(abstract, call("BurnMintTokenPool", "applyChainUpdates", ethMainnet)),
		batch(ethMainnet, call("BurnMintTokenPool", "applyChainUpdates", jovay, abstract)),
	}}

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	require.Len(t, anns, 2)
}

func TestAnalyze_IgnoresNonTokenPool(t *testing.T) {
	t.Parallel()

	proposal := symmetricProposalWithType("ERC20", seiMainnet, avaxMainnet)

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	assert.Empty(t, anns)
}

func TestAnalyze_IgnoresNonApplyChainUpdates(t *testing.T) {
	t.Parallel()

	proposal := &stubProposal{batches: []analyzer.DecodedBatchOperation{
		batch(seiMainnet, call("BurnMintTokenPool", "setRateLimiterAdmin", avaxMainnet)),
		batch(avaxMainnet, call("BurnMintTokenPool", "setRateLimiterAdmin", seiMainnet)),
	}}

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, proposal)
	require.NoError(t, err)
	assert.Empty(t, anns)
}

func TestAnalyze_EmptyProposal(t *testing.T) {
	t.Parallel()

	anns, err := (&LaneDetectorAnalyzer{}).Analyze(t.Context(), analyzer.ProposalAnalyzeRequest{}, &stubProposal{})
	require.NoError(t, err)
	assert.Empty(t, anns)
}

func symmetricProposal(chainA, chainB uint64) *stubProposal {
	return symmetricProposalWithType("BurnMintTokenPool", chainA, chainB)
}

func symmetricProposalWithType(contractType string, chainA, chainB uint64) *stubProposal {
	return &stubProposal{batches: []analyzer.DecodedBatchOperation{
		batch(chainA, call(contractType, "applyChainUpdates", chainB)),
		batch(chainB, call(contractType, "applyChainUpdates", chainA)),
	}}
}

func batch(chainSel uint64, calls ...analyzer.DecodedCall) *stubBatch {
	return &stubBatch{chainSelector: chainSel, calls: calls}
}

func call(contractType, method string, remoteSelectors ...uint64) analyzer.DecodedCall {
	updates := make([]token_pool.TokenPoolChainUpdate, len(remoteSelectors))
	for i, sel := range remoteSelectors {
		updates[i] = token_pool.TokenPoolChainUpdate{
			RemoteChainSelector:       sel,
			RemotePoolAddresses:       [][]byte{},
			OutboundRateLimiterConfig: token_pool.RateLimiterConfig{Capacity: big.NewInt(0), Rate: big.NewInt(0)},
			InboundRateLimiterConfig:  token_pool.RateLimiterConfig{Capacity: big.NewInt(0), Rate: big.NewInt(0)},
		}
	}

	return &stubCall{
		contractType: contractType,
		name:         method,
		inputs: analyzer.DecodedParameters{
			&stubParam{name: "chainsToAdd", rawValue: updates},
		},
	}
}

type stubProposal struct {
	batches []analyzer.DecodedBatchOperation
}

func (s *stubProposal) BatchOperations() analyzer.DecodedBatchOperations { return s.batches }

type stubBatch struct {
	chainSelector uint64
	calls         []analyzer.DecodedCall
}

func (s *stubBatch) ChainSelector() uint64        { return s.chainSelector }
func (s *stubBatch) Calls() analyzer.DecodedCalls { return s.calls }

type stubCall struct {
	contractType string
	name         string
	inputs       analyzer.DecodedParameters
}

func (s *stubCall) To() string                         { return "" }
func (s *stubCall) Name() string                       { return s.name }
func (s *stubCall) Inputs() analyzer.DecodedParameters { return s.inputs }
func (s *stubCall) Outputs() analyzer.DecodedParameters {
	return nil
}
func (s *stubCall) Data() []byte                      { return nil }
func (s *stubCall) AdditionalFields() json.RawMessage { return nil }
func (s *stubCall) ContractType() string              { return s.contractType }
func (s *stubCall) ContractVersion() string           { return "" }

type stubParam struct {
	name     string
	rawValue any
}

func (s *stubParam) Name() string  { return s.name }
func (s *stubParam) Type() string  { return "" }
func (s *stubParam) Value() any    { return s.rawValue }
func (s *stubParam) RawValue() any { return s.rawValue }
