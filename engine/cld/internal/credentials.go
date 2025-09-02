package internal

import (
	"crypto/tls"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

// GetCredsForEnv returns the appropriate gRPC transport credentials based on the environment.
func GetCredsForEnv(env string) credentials.TransportCredentials {
	if env == environment.Local {
		return insecure.NewCredentials()
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
	})
}
