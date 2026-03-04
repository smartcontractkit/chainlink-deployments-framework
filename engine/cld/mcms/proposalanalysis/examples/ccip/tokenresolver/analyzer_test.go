package tokenresolver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

func TestCanAnalyze_MatchesTokenPoolContractTypes(t *testing.T) {
	t.Parallel()

	a := &TokenMetadataAnalyzer{}

	for _, ct := range []string{
		"LockReleaseTokenPool",
		"BurnMintTokenPool",
		"BurnFromMintTokenPool",
		"BurnWithFromMintTokenPool",
		"TokenPool",
	} {
		t.Run(ct, func(t *testing.T) {
			t.Parallel()
			call := &stubCall{contractType: ct}
			assert.True(t, a.CanAnalyze(t.Context(), emptyCallReq(), call))
		})
	}
}

func TestCanAnalyze_RejectsUnknownContractTypes(t *testing.T) {
	t.Parallel()

	a := &TokenMetadataAnalyzer{}

	for _, ct := range []string{"", "ERC20", "Router", "OnRamp", "OffRamp"} {
		t.Run(ct, func(t *testing.T) {
			t.Parallel()
			call := &stubCall{contractType: ct}
			assert.False(t, a.CanAnalyze(t.Context(), emptyCallReq(), call))
		})
	}
}

func TestCanAnalyze_MatchesAnyMethod(t *testing.T) {
	t.Parallel()

	a := &TokenMetadataAnalyzer{}
	call := &stubCall{contractType: "TokenPool", name: "anyMethodName"}
	assert.True(t, a.CanAnalyze(t.Context(), emptyCallReq(), call))
}

type stubCall struct {
	contractType    string
	contractVersion string
	name            string
	to              string
}

func (s *stubCall) To() string                        { return s.to }
func (s *stubCall) Name() string                      { return s.name }
func (s *stubCall) Inputs() decoder.DecodedParameters { return nil }
func (s *stubCall) Outputs() decoder.DecodedParameters {
	return nil
}
func (s *stubCall) Data() []byte                      { return nil }
func (s *stubCall) AdditionalFields() json.RawMessage { return nil }
func (s *stubCall) ContractType() string              { return s.contractType }
func (s *stubCall) ContractVersion() string           { return s.contractVersion }

var _ decoder.DecodedCall = (*stubCall)(nil)

func emptyCallReq() analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext] {
	return analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext]{}
}
