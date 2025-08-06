package remote

import (
	"fmt"

	datastore2 "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/remote/internal/protos"
)

func ThrowAndCatch(
	catalog *catalogDataStore,
	request *datastore2.DataAccessRequest,
) (*datastore2.DataAccessResponse, error) {
	// Create a bidirectional stream
	stream, err := catalog.client.DataAccess()
	if err != nil {
		return nil, fmt.Errorf("failed to create data access stream: %w", err)
	}
	if sendErr := stream.Send(request); sendErr != nil {
		return nil, fmt.Errorf("failed to send begin-transaction request: %w", sendErr)
	}
	// Receive the response
	response, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}
	return response, nil
}
