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

// MockJDClient is a composite client that implements the offchain.Client interface
// by embedding the individual mock service clients
type MockJDClient struct {
	*MockJobServiceClient
	*MockNodeServiceClient
	*MockCSAServiceClient
}

// NewMockJDClient creates a new MockJDClient.
func NewMockJDClient(t *testing.T) *MockJDClient {
	t.Helper()

	mockClients := NewMockClients(t)

	return &MockJDClient{
		MockJobServiceClient:  mockClients.JobServiceClient,
		MockNodeServiceClient: mockClients.NodeServiceClient,
		MockCSAServiceClient:  mockClients.CSAServiceClient,
	}
}
