package remote

import (
	"fmt"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	rpb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// checkResponseStatus converts a ResponseStatus to a standard gRPC error.
// Returns nil if the status indicates success (code == 0).
// Preserves error details for rich error information.
//
// Usage:
//
//	if err := checkResponseStatus(resp.Status); err != nil {
//	    st, _ := status.FromError(err)
//	    log.Printf("Error: %v (code: %v)", st.Message(), st.Code())
//	}
func checkResponseStatus(rs *pb.ResponseStatus) error {
	if rs == nil {
		return status.Error(codes.Internal, "missing response status")
	}

	// Success case
	if rs.Code == 0 {
		return nil
	}

	// Convert to google.rpc.Status, then to gRPC status
	// This preserves the details field!
	st := status.FromProto(&rpb.Status{
		Code:    rs.Code,
		Message: rs.Message,
		Details: rs.Details,
	})

	return st.Err()
}
