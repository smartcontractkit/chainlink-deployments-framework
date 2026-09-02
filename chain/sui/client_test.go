package sui

import (
	"net/url"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrpcTargetFromNodeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		nodeURL         string
		want            string
		wantErr         string
		wantNotContains string // asserted in the error string when wantErr is set
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
		{
			name:    "userinfo is stripped from gRPC target",
			nodeURL: "https://some-secret@sui-testnet.g.alchemy.com",
			want:    "sui-testnet.g.alchemy.com:443",
		},
		{
			name:            "userinfo is redacted from missing-host error",
			nodeURL:         "https://super-secret-key@/path",
			wantErr:         "has no host",
			wantNotContains: "super-secret-key",
		},
		{
			name:            "userinfo is redacted from parse-error message",
			nodeURL:         "https://super-secret-key@bad\x7f",
			wantErr:         "parse node URL",
			wantNotContains: "super-secret-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := grpcTargetFromNodeURL(tt.nodeURL)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantNotContains != "" {
					assert.NotContains(t, err.Error(), tt.wantNotContains)
				}

				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGrpcTargetFromNodeURL_ParseErrorPreservesUnwrapChain(t *testing.T) {
	t.Parallel()

	// The parse-error path wraps a sanitized *url.Error with %w, so the unwrap chain must remain
	// intact (errors.As reaches *url.Error) while the raw userinfo never leaks into the error
	// string or the wrapped url.Error.URL.
	_, err := grpcTargetFromNodeURL("https://super-secret-key@bad\x7f")
	require.Error(t, err)

	var ue *url.Error
	require.ErrorAs(t, err, &ue, "parse error should unwrap to *url.Error")

	assert.NotContains(t, ue.URL, "super-secret-key")
	assert.NotContains(t, err.Error(), "super-secret-key")
}

func TestRedactURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "userinfo username stripped",
			raw:  "https://alchemy-key@sui-testnet.g.alchemy.com",
			want: "https://sui-testnet.g.alchemy.com",
		},
		{
			name: "userinfo username and password stripped",
			raw:  "https://user:pass@sui-testnet.g.alchemy.com",
			want: "https://sui-testnet.g.alchemy.com",
		},
		{
			name: "no userinfo unchanged",
			raw:  "https://sui-testnet.g.alchemy.com:443/path?q=1",
			want: "https://sui-testnet.g.alchemy.com:443/path?q=1",
		},
		{
			name: "unparseable URL with userinfo best-effort redacted",
			raw:  "https://secret-key@bad\x7f",
			want: "<redacted>@bad\x7f",
		},
		{
			name: "unparseable URL without userinfo returned as-is",
			raw:  "http://bad\x7f",
			want: "http://bad\x7f",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, redactURL(tt.raw))
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

	t.Run("URL with userinfo and empty explicit token constructs client", func(t *testing.T) {
		t.Parallel()

		// The token is read from userinfo (legacy fallback path); construction must not dial,
		// so this works without a live node.
		client, err := NewPTBClientFromNodeURL(log, "https://some-secret@127.0.0.1:9000", "")
		require.NoError(t, err)
		require.NotNil(t, client)
	})
}

func TestGrpcTokenFromNodeURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nodeURL  string
		explicit string
		want     string
	}{
		{
			name:     "explicit token wins over userinfo",
			nodeURL:  "https://userinfo-secret@sui-testnet.g.alchemy.com",
			explicit: "explicit-token",
			want:     "explicit-token",
		},
		{
			name:     "explicit token wins when URL has no userinfo",
			nodeURL:  "https://sui-testnet.g.alchemy.com",
			explicit: "explicit-token",
			want:     "explicit-token",
		},
		{
			name:     "userinfo used when explicit empty",
			nodeURL:  "https://alchemy-key@sui-testnet.g.alchemy.com",
			explicit: "",
			want:     "alchemy-key",
		},
		{
			name:     "default token when neither explicit nor userinfo",
			nodeURL:  "http://127.0.0.1:9000",
			explicit: "",
			want:     defaultGrpcToken,
		},
		{
			name:     "empty userinfo username falls through to default",
			nodeURL:  "https://@sui-testnet.g.alchemy.com",
			explicit: "",
			want:     defaultGrpcToken,
		},
		{
			name:     "unparseable URL with empty explicit falls through to default",
			nodeURL:  "http://\x7f",
			explicit: "",
			want:     defaultGrpcToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := grpcTokenFromNodeURL(tt.nodeURL, tt.explicit)
			assert.Equal(t, tt.want, got)
		})
	}
}
