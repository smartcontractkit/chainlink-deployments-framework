package cre_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/cre"
	cremocks "github.com/smartcontractkit/chainlink-deployments-framework/cre/mocks"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	cli := cremocks.NewMockCLIRunner(t)
	var c stubClient

	tests := []struct {
		name       string
		opts       []cre.RunnerOption
		wantNil    bool
		wantCLI    cre.CLIRunner
		wantClient cre.Client
	}{
		{
			name:       "both options",
			opts:       []cre.RunnerOption{cre.WithCLI(cli), cre.WithClient(&c)},
			wantCLI:    cli,
			wantClient: &c,
		},
		{
			name:    "cli only",
			opts:    []cre.RunnerOption{cre.WithCLI(cli)},
			wantCLI: cli,
		},
		{
			name:       "client only",
			opts:       []cre.RunnerOption{cre.WithClient(&c)},
			wantClient: &c,
		},
		{
			name: "no options",
		},
		{
			name:    "nil interface value",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.wantNil {
				var r cre.Runner
				require.Nil(t, r)

				return
			}

			r := cre.NewRunner(tt.opts...)
			require.NotNil(t, r)
			require.Equal(t, tt.wantCLI, r.CLI())
			require.Equal(t, tt.wantClient, r.Client())
		})
	}
}

// stubClient implements [Client] for tests (empty interface: any concrete type will do).
type stubClient struct{}
