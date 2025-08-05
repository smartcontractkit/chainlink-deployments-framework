package offchain

import "context"

// Provider interface for offchain client providers.
type Provider interface {
	// Initialize sets up the offchain client and returns the instance.
	Initialize(ctx context.Context) (OffchainClient, error)
	// Name returns a human-readable name for this provider.
	Name() string
	// OffchainClient returns the initialized offchain client instance.
	// You must call Initialize before using this method.
	OffchainClient() OffchainClient
}
