package remote

import (
	"errors"
	"fmt"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	rpb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseResponseStatus converts a ResponseStatus to a standard gRPC error.
// Returns nil if the status indicates success (code == 0).
// Preserves error details for rich error information.
//
// Usage:
//
//	if err := parseResponseStatus(resp.Status); err != nil {
//	    st, _ := status.FromError(err)
//	    log.Printf("Error: %v (code: %v)", st.Message(), st.Code())
//	}
func parseResponseStatus(rs *pb.ResponseStatus) error {
	if rs == nil {
		return status.Error(codes.Internal, "missing response status")
	}

	// Success case
	if rs.Code == 0 {
		return nil
	}

	// Convert to google.rpc.Status, then to gRPC status
	st := status.FromProto(&rpb.Status{
		Code:    rs.Code,
		Message: rs.Message,
		Details: rs.Details,
	})

	return st.Err()
}

// parseStatusError converts a gRPC error to a standard gRPC status.
func parseStatusError(err error) (*status.Status, error) {
	if err == nil {
		return nil, errors.New("nil error provided")
	}

	st, ok := status.FromError(err)
	if !ok {
		return nil, fmt.Errorf("failed to parse error: %s", err.Error())
	}

	return st, nil
}
