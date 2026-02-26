package decoder

import (
	"context"
	"errors"
	"testing"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

func TestNewExperimentalDecoder(t *testing.T) {
	t.Parallel()

	t.Run("creates decoder with config", func(t *testing.T) {
		t.Parallel()

		cfg := Config{
			EVMABIMappings: map[string]string{"Router 1.0.0": `[{"type":"function"}]`},
		}
		dec := NewExperimentalDecoder(cfg)

		require.NotNil(t, dec)
		assert.Equal(t, cfg.EVMABIMappings, dec.config.EVMABIMappings)
		assert.NotNil(t, dec.buildReport)
	})

	t.Run("creates decoder with empty config", func(t *testing.T) {
		t.Parallel()

		dec := NewExperimentalDecoder(Config{})

		require.NotNil(t, dec)
		assert.NotNil(t, dec.buildReport)
	})
}

func TestExperimentalDecoderDecode(t *testing.T) {
	t.Parallel()

	t.Run("nil proposal returns error", func(t *testing.T) {
		t.Parallel()

		dec := newTestDecoder(stubBuilder(&experimentalanalyzer.ProposalReport{}, nil))

		result, err := dec.Decode(t.Context(), deployment.Environment{}, nil)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "proposal cannot be nil")
	})

	t.Run("report builder error is propagated", func(t *testing.T) {
		t.Parallel()

		dec := newTestDecoder(stubBuilder(nil, errors.New("decode failed: chain unavailable")))

		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{ChainSelector: 1, Transactions: []mcmstypes.Transaction{{To: "0x1"}}},
			},
		}

		result, err := dec.Decode(t.Context(), deployment.Environment{}, proposal)

		require.Error(t, err)
		assert.Nil(t, result)
		require.ErrorContains(t, err, "building timelock report")
		require.ErrorContains(t, err, "chain unavailable")
	})

	t.Run("nil report returns error", func(t *testing.T) {
		t.Parallel()

		dec := newTestDecoder(stubBuilder(nil, nil))
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{ChainSelector: 1, Transactions: []mcmstypes.Transaction{{To: "0x1"}}},
			},
		}

		result, err := dec.Decode(t.Context(), deployment.Environment{}, proposal)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "report builder returned a nil report")
	})

	t.Run("successful decode converts report to decoded types", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{
					ChainSelector: 1111,
					ChainName:     "ethereum",
					Operations: []experimentalanalyzer.OperationReport{
						{
							Calls: []*experimentalanalyzer.DecodedCall{
								{
									Address:         "0xRouter",
									Method:          "function setConfig(uint256)",
									ContractType:    "Router",
									ContractVersion: "2.0.0",
									Inputs: []experimentalanalyzer.NamedField{
										{Name: "val", Value: experimentalanalyzer.SimpleField{Value: "100"}, RawValue: "100"},
									},
								},
							},
						},
					},
				},
			},
		}

		dec := newTestDecoder(stubBuilder(report, nil))

		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{
					ChainSelector: 1111,
					Transactions: []mcmstypes.Transaction{
						{
							To:               "0xRouter",
							Data:             []byte{0xAB},
							AdditionalFields: nil,
							OperationMetadata: mcmstypes.OperationMetadata{
								ContractType: "Router",
							},
						},
					},
				},
			},
		}

		result, err := dec.Decode(t.Context(), deployment.Environment{}, proposal)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.BatchOperations(), 1)

		batch := result.BatchOperations()[0]
		assert.Equal(t, uint64(1111), batch.ChainSelector())

		require.Len(t, batch.Calls(), 1)

		call := batch.Calls()[0]
		assert.Equal(t, "0xRouter", call.To())
		assert.Equal(t, "setConfig", call.Name())
		assert.Equal(t, "Router", call.ContractType())
		assert.Equal(t, "2.0.0", call.ContractVersion())
		assert.Equal(t, []byte{0xAB}, call.Data())
	})

	t.Run("empty report produces empty result", func(t *testing.T) {
		t.Parallel()

		dec := newTestDecoder(stubBuilder(&experimentalanalyzer.ProposalReport{}, nil))

		result, err := dec.Decode(t.Context(), deployment.Environment{}, &mcms.TimelockProposal{})

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Empty(t, result.BatchOperations())
	})

	t.Run("multiple batches are fully converted", func(t *testing.T) {
		t.Parallel()

		report := &experimentalanalyzer.ProposalReport{
			Batches: []experimentalanalyzer.BatchReport{
				{
					ChainSelector: 100,
					ChainName:     "chain-a",
					Operations: []experimentalanalyzer.OperationReport{
						{Calls: []*experimentalanalyzer.DecodedCall{{Address: "0x1", Method: "foo"}}},
					},
				},
				{
					ChainSelector: 200,
					ChainName:     "chain-b",
					Operations: []experimentalanalyzer.OperationReport{
						{Calls: []*experimentalanalyzer.DecodedCall{{Address: "0x2", Method: "bar"}}},
					},
				},
			},
		}

		dec := newTestDecoder(stubBuilder(report, nil))
		proposal := &mcms.TimelockProposal{
			Operations: []mcmstypes.BatchOperation{
				{ChainSelector: 100, Transactions: []mcmstypes.Transaction{{To: "0x1"}}},
				{ChainSelector: 200, Transactions: []mcmstypes.Transaction{{To: "0x2"}}},
			},
		}

		result, err := dec.Decode(t.Context(), deployment.Environment{}, proposal)

		require.NoError(t, err)
		require.Len(t, result.BatchOperations(), 2)
		assert.Equal(t, uint64(100), result.BatchOperations()[0].ChainSelector())
		assert.Equal(t, uint64(200), result.BatchOperations()[1].ChainSelector())
		assert.Equal(t, "foo", result.BatchOperations()[0].Calls()[0].Name())
		assert.Equal(t, "bar", result.BatchOperations()[1].Calls()[0].Name())
	})
}

// newTestDecoder creates an ExperimentalDecoder with a custom report builder.
func newTestDecoder(builder reportBuilderFunc) *ExperimentalDecoder {
	return &ExperimentalDecoder{
		config:      Config{},
		buildReport: builder,
	}
}

func stubBuilder(
	report *experimentalanalyzer.ProposalReport,
	err error,
) reportBuilderFunc {
	return func(
		_ context.Context,
		_ deployment.Environment,
		_ *mcms.TimelockProposal,
	) (*experimentalanalyzer.ProposalReport, error) {
		return report, err
	}
}
