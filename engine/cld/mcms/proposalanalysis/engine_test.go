package proposalanalysis

import (
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	experimentalanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestEngineWithLogger(t *testing.T) {
	t.Run("engine accepts logger from options", func(t *testing.T) {
		lggr := logger.Test(t)
		engine := NewAnalyzerEngine(WithLogger(lggr))

		assert.NotNil(t, engine)
		// Verify the logger is set by checking the concrete type
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)
		assert.NotNil(t, concreteEngine.logger)
	})

	t.Run("engine uses nop logger when not provided", func(t *testing.T) {
		engine := NewAnalyzerEngine()

		assert.NotNil(t, engine)
		// Verify the logger is set (will be Nop logger)
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)
		assert.NotNil(t, concreteEngine.logger)
	})
}

func TestEngineRegistryRegistration(t *testing.T) {
	t.Run("register EVM ABI mappings", func(t *testing.T) {
		engine := NewAnalyzerEngine()
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)

		err := concreteEngine.RegisterEVMABIMappings(map[string]string{
			"MyContract 1.0.0": `[{"type":"function","name":"f","inputs":[]}]`,
		})
		require.NoError(t, err)
		assert.Len(t, concreteEngine.evmABIMappings, 1)
	})

	t.Run("reject duplicate EVM ABI mappings", func(t *testing.T) {
		engine := NewAnalyzerEngine()
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)

		firstErr := concreteEngine.RegisterEVMABIMappings(map[string]string{
			"MyContract 1.0.0": `[{"type":"function","name":"f","inputs":[]}]`,
		})
		require.NoError(t, firstErr)

		err := concreteEngine.RegisterEVMABIMappings(map[string]string{
			"MyContract 1.0.0": `[{"type":"function","name":"g","inputs":[]}]`,
		})
		require.Error(t, err)
		assert.Equal(t, `evm ABI mapping for key "MyContract 1.0.0" is already registered`, err.Error())
	})

	t.Run("register Solana decoders", func(t *testing.T) {
		engine := NewAnalyzerEngine()
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)

		err := concreteEngine.RegisterSolanaDecoders(map[string]experimentalanalyzer.DecodeInstructionFn{
			"MyProgram 1.0.0": func(_ []*solana.AccountMeta, _ []byte) (experimentalanalyzer.AnchorInstruction, error) {
				return nil, nil
			},
		})
		require.NoError(t, err)
		assert.Len(t, concreteEngine.solanaDecoders, 1)
	})

	t.Run("reject duplicate Solana decoders", func(t *testing.T) {
		engine := NewAnalyzerEngine()
		concreteEngine, ok := engine.(*analyzerEngine)
		require.True(t, ok)

		firstErr := concreteEngine.RegisterSolanaDecoders(map[string]experimentalanalyzer.DecodeInstructionFn{
			"MyProgram 1.0.0": func(_ []*solana.AccountMeta, _ []byte) (experimentalanalyzer.AnchorInstruction, error) {
				return nil, nil
			},
		})
		require.NoError(t, firstErr)

		err := concreteEngine.RegisterSolanaDecoders(map[string]experimentalanalyzer.DecodeInstructionFn{
			"MyProgram 1.0.0": func(_ []*solana.AccountMeta, _ []byte) (experimentalanalyzer.AnchorInstruction, error) {
				return nil, nil
			},
		})
		require.Error(t, err)
		assert.Equal(t, `solana decoder for key "MyProgram 1.0.0" is already registered`, err.Error())
	})
}
