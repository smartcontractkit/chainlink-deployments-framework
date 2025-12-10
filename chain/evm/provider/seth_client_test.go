package provider

import (
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
func Test_newSethClient(t *testing.T) {
	var (
		chainID    = chainsel.TEST_1000.EvmChainID
		configPath = writeSethConfigFile(t)
	)

	// Create a mock RPC server that always returns a valid response for eth_blockNumber
	mockSrv := newFakeRPCServer(t)

	tests := []struct {
		name                string
		giveRPCURL          string
		giveChainID         uint64
		giveConfigFilePath  string
		giveGethWrapperDirs []string
		wantErr             string
	}{
		{
			name:                "valid using defaults",
			giveRPCURL:          mockSrv.URL,
			giveChainID:         chainID,
			giveConfigFilePath:  "",
			giveGethWrapperDirs: []string{},
		},
		{
			name:                "valid with config file",
			giveRPCURL:          mockSrv.URL,
			giveChainID:         chainID,
			giveConfigFilePath:  configPath,
			giveGethWrapperDirs: []string{},
		},
		{
			name:                "error reading config file",
			giveRPCURL:          mockSrv.URL,
			giveChainID:         chainID,
			giveConfigFilePath:  "nonexistent.toml",
			giveGethWrapperDirs: []string{},
			wantErr:             "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // This test cannot run in parallel due to a race condition in seth's log initialization
			got, err := newSethClient(
				tt.giveRPCURL, tt.giveChainID, tt.giveGethWrapperDirs, tt.giveConfigFilePath,
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

func Test_readSethConfigFromFile(t *testing.T) {
	t.Parallel()

	validConfigPath := writeSethConfigFile(t)
	invalidConfigPath := writeInvalidSethConfigFile(t)

	tests := []struct {
		name             string
		path             string
		wantArtifactsDir string
		wantErr          string
	}{
		{
			name:             "success",
			path:             validConfigPath,
			wantArtifactsDir: "artifacts",
		},
		{
			name:    "file not found",
			path:    "nonexistent.toml",
			wantErr: "no such file or directory",
		},
		{
			name:    "invalid toml",
			path:    invalidConfigPath,
			wantErr: "unable to unmarshal seth config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := readSethConfigFromFile(tt.path)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.Equal(t, tt.wantArtifactsDir, cfg.ArtifactsDir)
			}
		})
	}
}
