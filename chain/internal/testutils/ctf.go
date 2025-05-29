package testutils

import "sync"

var (
	// DefaultNetworkOnce is a sync.Once instance that ensures the CTF framework only sets up the
	// DefaultNetwork once.
	DefaultNetworkOnce = &sync.Once{}
)
