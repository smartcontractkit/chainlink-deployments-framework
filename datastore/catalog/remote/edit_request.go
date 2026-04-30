package remote

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

// errorMapper translates a non-nil gRPC status error into a domain-specific
// error. Receives the raw status error and its extracted code for switch-based
// mapping. If nil, executeEdit falls back to: fmt.Errorf("%s failed: %w", opName, statusErr).
type errorMapper func(statusErr error, code codes.Code) error

// executeEdit handles the open-stream / send / recv / status-check sequence
// shared by every *EditRequest call. R must be the concrete edit-response
// pointer type (e.g. *pb.AddressReferenceEditResponse); extract returns the
// typed response from the DataAccessResponse oneof and executeEdit returns
// "unexpected response type" when it is nil.
func executeEdit[R comparable](
	client *CatalogClient,
	req *pb.DataAccessRequest,
	opName string,
	extract func(*pb.DataAccessResponse) R,
	mapErr errorMapper,
) error {
	stream, err := client.DataAccess(req)
	if err != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", err)
	}

	if sendErr := stream.Send(req); sendErr != nil {
		return fmt.Errorf("failed to send %s request: %w", opName, sendErr)
	}

	resp, recvErr := stream.Recv()
	if recvErr != nil {
		if errors.Is(recvErr, io.EOF) {
			return errors.New("unexpected end of stream")
		}

		return fmt.Errorf("failed to receive %s response: %w", opName, recvErr)
	}

	if statusErr := parseResponseStatus(resp.Status); statusErr != nil {
		if mapErr != nil {
			st, parseErr := parseStatusError(statusErr)
			if parseErr != nil {
				return parseErr
			}

			return mapErr(statusErr, st.Code())
		}

		return fmt.Errorf("%s failed: %w", opName, statusErr)
	}

	var zero R
	if extract(resp) == zero {
		return errors.New("unexpected response type")
	}

	return nil
}
