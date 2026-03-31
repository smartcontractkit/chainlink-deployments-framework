package cre

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	t.Parallel()

	cli := NewCLIRunner("/bin/sh", "")
	var c stubClient

	tests := []struct {
		name       string
		opts       []RunnerOption
		wantNil    bool
		wantCLI    CLIRunner
		wantClient Client
	}{
		{
			name:       "both options",
			opts:       []RunnerOption{WithCLI(cli), WithClient(&c)},
			wantCLI:    cli,
			wantClient: &c,
		},
		{
			name:    "cli only",
			opts:    []RunnerOption{WithCLI(cli)},
			wantCLI: cli,
		},
		{
			name:       "client only",
			opts:       []RunnerOption{WithClient(&c)},
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
				var r Runner
				require.Nil(t, r)

				return
			}

			r := NewRunner(tt.opts...)
			require.NotNil(t, r)
			require.Equal(t, tt.wantCLI, r.CLI())
			require.Equal(t, tt.wantClient, r.Client())
		})
	}
}

// stubClient implements [Client] for tests (empty interface: any concrete type will do).
type stubClient struct{}
