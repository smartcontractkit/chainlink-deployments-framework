package credentials

import (
	"crypto/tls"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// GetCredsForEnv returns the appropriate gRPC transport credentials based on the environment.
func GetCredsForEnv(env string) credentials.TransportCredentials {
	if env == "local" {
		return insecure.NewCredentials()
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
	})
}
