// Package memory provides an in-memory implementation of the Job Distributor client.
//
// This package is designed for testing purposes when you don't need a real JD backend.
// All data is stored in memory and is lost when the client is garbage collected.
//
// # Thread Safety
//
// The MemoryJobDistributor implementation is thread-safe and uses sync.RWMutex to protect
// concurrent access to internal data structures. Multiple goroutines can safely call methods
// on the same MemoryJobDistributor instance concurrently.
//
// # Usage
//
// Create a new in-memory JD client:
//
//	client := memory.NewMemoryJobDistributor()
//
// Use it like any other JD client:
//
//	resp, err := client.ProposeJob(ctx, &jobv1.ProposeJobRequest{
//	    NodeId: "test-node",
//	    Spec:   "job spec here",
//	})
//
// # Limitations
//
// - No persistence: All data is stored in memory only
// - Test-only: This implementation is intended for testing and should not be used in production environments
//
// # Implementation Details
//
// The client maintains four in-memory maps:
//   - jobs: Stores job instances by job ID
//   - proposals: Stores job proposals by proposal ID
//   - nodes: Stores node registrations by node ID
//   - keypairs: Stores CSA keypairs by public key
//
// All write operations (create, update, delete) modify these maps directly,
// and read operations return data from these maps.
package memory
