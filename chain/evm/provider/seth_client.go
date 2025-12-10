package provider

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/smartcontractkit/chainlink-testing-framework/seth"
)

// newSethClient creates a new Seth client with the provided configuration. The Seth client is used
// to interact with EVM chains, providing tracing and error handling.
func newSethClient(
	rpcURL string,
	chainID uint64,
	gethWrapperDirs []string,
	configFilePath string,
) (*seth.Client, error) {
	if configFilePath != "" {
		sethConfig, readErr := readSethConfigFromFile(configFilePath)
		if readErr != nil {
			return nil, readErr
		}

		return seth.NewClientBuilderWithConfig(sethConfig).
			UseNetworkWithChainId(chainID).
			WithRpcUrl(rpcURL).
			WithReadOnlyMode().
			Build()
	}

	// If configuration is not needed we create a client with reasonable defaults.
	// if you need to further tweak them, please refer to
	// https://github.com/smartcontractkit/chainlink-testing-framework/blob/main/seth/README.md
	return seth.NewClientBuilder().
		WithRpcUrl(rpcURL).
		WithGethWrappersFolders(gethWrapperDirs).
		WithTracing(seth.TracingLevel_Reverted, []string{seth.TraceOutput_Console, seth.TraceOutput_JSON, seth.TraceOutput_DOT}).
		WithReadOnlyMode().
		Build()
}

// readSethConfigFromFile reads the Seth configuration from a TOML file and returns a seth.Config.
func readSethConfigFromFile(configPath string) (*seth.Config, error) {
	d, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var sethConfig seth.Config
	if err = toml.Unmarshal(d, &sethConfig); err != nil {
		return nil, fmt.Errorf("unable to unmarshal seth config: %w", err)
	}

	return &sethConfig, nil
}
