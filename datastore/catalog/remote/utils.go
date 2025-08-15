package remote

import (
	"fmt"

	pb "github.com/smartcontractkit/chainlink-protos/chainlink-catalog/v1/datastore"
)

func ThrowAndCatch(
	catalog *catalogDataStore,
	request *pb.DataAccessRequest,
) (*pb.DataAccessResponse, error) {
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
