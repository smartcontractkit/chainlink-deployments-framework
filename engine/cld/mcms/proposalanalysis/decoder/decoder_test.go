package decoder_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/stretchr/testify/require"
)

// TestDecoderOptions verifies that decoder options work correctly
func TestDecoderOptions(t *testing.T) {
	t.Run("can create decoder with no options", func(t *testing.T) {
		d := decoder.NewLegacyDecoder()
		require.NotNil(t, d)
	})

	t.Run("can inject custom proposal context", func(t *testing.T) {
		customContext := &mockProposalContext{}

		d := decoder.NewLegacyDecoder(
			decoder.WithProposalContext(customContext),
		)
		require.NotNil(t, d)
	})
}

// mockProposalContext is a minimal mock for testing
type mockProposalContext struct{}

func (m *mockProposalContext) GetEVMRegistry() experimentalanalyzer.EVMABIRegistry {
	return nil
}

func (m *mockProposalContext) GetSolanaDecoderRegistry() experimentalanalyzer.SolanaDecoderRegistry {
	return nil
}

func (m *mockProposalContext) FieldsContext(chainSelector uint64) *experimentalanalyzer.FieldContext {
	return nil
}

func (m *mockProposalContext) GetRenderer() experimentalanalyzer.Renderer {
	return nil
}

func (m *mockProposalContext) SetRenderer(renderer experimentalanalyzer.Renderer) {
	// no-op
}
