package mocks

import "testing"

// MockClients is a mock of the Job Distributor protobuf clients.
type MockClients struct {
	CSAServiceClient  *MockCSAServiceClient
	JobServiceClient  *MockJobServiceClient
	NodeServiceClient *MockNodeServiceClient
}

// NewMockClients creates a new MockClients.
func NewMockClients(t *testing.T) *MockClients {
	t.Helper()

	return &MockClients{
		CSAServiceClient:  NewMockCSAServiceClient(t),
		JobServiceClient:  NewMockJobServiceClient(t),
		NodeServiceClient: NewMockNodeServiceClient(t),
	}
}
