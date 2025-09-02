package internal

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

func TestGetCredsForEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		env          string
		wantInsecure bool
		wantTLS      bool
	}{
		{
			name:         "local environment returns insecure credentials",
			env:          environment.Local,
			wantInsecure: true,
			wantTLS:      false,
		},
		{
			name:         "empty environment returns TLS credentials",
			env:          "",
			wantInsecure: false,
			wantTLS:      true,
		},
		{
			name:         "non local environment returns TLS credentials",
			env:          "custom-env",
			wantInsecure: false,
			wantTLS:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			creds := GetCredsForEnv(tt.env)
			require.NotNil(t, creds)

			if tt.wantInsecure {
				// For insecure credentials, check that it's the insecure type
				insecureCreds := insecure.NewCredentials()
				assert.IsType(t, insecureCreds, creds)
			}

			if tt.wantTLS {
				// For TLS credentials, verify it's a TLS type and check configuration
				info := creds.Info()
				assert.Equal(t, "tls", info.SecurityProtocol)

				// Create a reference TLS credential to compare behavior
				expectedCreds := credentials.NewTLS(&tls.Config{
					MinVersion: tls.VersionTLS12,
				})
				expectedInfo := expectedCreds.Info()

				// Both should have the same security protocol and server name behavior
				assert.Equal(t, expectedInfo.SecurityProtocol, info.SecurityProtocol)
				assert.Equal(t, expectedInfo.ServerName, info.ServerName)
			}
		})
	}
}
