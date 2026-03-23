package artifacts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
)

func TestHTTPClientFromCREConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cre     cfgenv.CREConfig
		wantNil bool
		wantDur time.Duration
	}{
		{name: "empty_timeout", cre: cfgenv.CREConfig{}, wantNil: true},
		{name: "valid_timeout", cre: cfgenv.CREConfig{Timeout: "30s"}, wantNil: false, wantDur: 30 * time.Second},
		{name: "invalid_timeout", cre: cfgenv.CREConfig{Timeout: "not-a-duration"}, wantNil: true},
		{name: "whitespace_only", cre: cfgenv.CREConfig{Timeout: "   "}, wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := HTTPClientFromCREConfig(tt.cre)
			if tt.wantNil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.wantDur, got.Timeout)
		})
	}
}
