package remote

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc/codes"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

func getOpName(req *pb.DataAccessRequest) (string, error) {
	var op, entity string
	var err error
	switch r := req.Operation.(type) {
	case *pb.DataAccessRequest_AddressReferenceEditRequest:
		entity = "address ref"
		op, err = semanticsLabel(r.AddressReferenceEditRequest.Semantics)
		if err != nil {
			return "", err
		}
	case *pb.DataAccessRequest_ChainMetadataEditRequest:
		entity = "chain metadata"
		op, err = semanticsLabel(r.ChainMetadataEditRequest.Semantics)
		if err != nil {
			return "", err
		}
	case *pb.DataAccessRequest_ContractMetadataEditRequest:
		entity = "contract metadata"
		op, err = semanticsLabel(r.ContractMetadataEditRequest.Semantics)
		if err != nil {
			return "", err
		}
	case *pb.DataAccessRequest_EnvironmentMetadataEditRequest:
		entity = "env metadata"
		op, err = semanticsLabel(r.EnvironmentMetadataEditRequest.Semantics)
		if err != nil {
			return "", err
		}
	default:
		return "", errors.New("unknown operation type")
	}

	return op + " " + entity, nil
}

func semanticsLabel(s pb.EditSemantics) (string, error) {
	switch s {
	case pb.EditSemantics_SEMANTICS_INSERT:
		return "add", nil
	case pb.EditSemantics_SEMANTICS_UPSERT:
		return "upsert", nil
	case pb.EditSemantics_SEMANTICS_UPDATE:
		return "update", nil
	case pb.EditSemantics_SEMANTICS_DELETE:
		return "delete", nil
	default:
		return "", errors.New("unknown semantics")
	}
}

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
	extract func(*pb.DataAccessResponse) R,
	mapErr errorMapper,
) error {
	opName, err := getOpName(req)
	if err != nil {
		return fmt.Errorf("failed to get operation name: %w", err)
	}
	stream, clientErr := client.DataAccess(req)
	if clientErr != nil {
		return fmt.Errorf("failed to create gRPC stream: %w", clientErr)
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
