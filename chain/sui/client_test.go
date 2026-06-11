package sui

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcTargetFromNodeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		nodeURL string
		want    string
		wantErr string
	}{
		{
			name:    "host with explicit port",
			nodeURL: "http://127.0.0.1:9000",
			want:    "127.0.0.1:9000",
		},
		{
			name:    "http scheme defaults to port 9000",
			nodeURL: "http://example.com",
			want:    "example.com:9000",
		},
		{
			name:    "https scheme defaults to port 443",
			nodeURL: "https://example.com",
			want:    "example.com:443",
		},
		{
			name:    "ipv6 host with port gets bracketed",
			nodeURL: "http://[::1]:9000",
			want:    "[::1]:9000",
		},
		{
			name:    "ipv6 host without port gets bracketed and defaulted",
			nodeURL: "http://[::1]",
			want:    "[::1]:9000",
		},
		{
			name:    "invalid URL returns error",
			nodeURL: "http://\x7f",
			wantErr: "parse node URL",
		},
		{
			name:    "missing host returns error",
			nodeURL: "http:///path",
			wantErr: "has no host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := grpcTargetFromNodeURL(tt.nodeURL)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewPTBClientFromNodeURL(t *testing.T) {
	t.Parallel()

	log, err := logger.New()
	require.NoError(t, err)

	t.Run("valid URL with empty token uses default", func(t *testing.T) {
		t.Parallel()

		client, err := NewPTBClientFromNodeURL(log, "http://127.0.0.1:9000", "")
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("valid URL with explicit token", func(t *testing.T) {
		t.Parallel()

		client, err := NewPTBClientFromNodeURL(log, "http://127.0.0.1:9000", "my-token")
		require.NoError(t, err)
		require.NotNil(t, client)
	})

	t.Run("invalid URL returns error", func(t *testing.T) {
		t.Parallel()

		client, err := NewPTBClientFromNodeURL(log, "http://\x7f", "")
		require.Error(t, err)
		require.Nil(t, client)
	})
}
