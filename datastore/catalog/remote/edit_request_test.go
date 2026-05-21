package remote

import (
	"testing"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"github.com/stretchr/testify/require"
)

func TestGetOpName(t *testing.T) {
	t.Parallel()

	addressRefOp := func(s pb.EditSemantics) *pb.DataAccessRequest {
		return &pb.DataAccessRequest{
			Operation: &pb.DataAccessRequest_AddressReferenceEditRequest{
				AddressReferenceEditRequest: &pb.AddressReferenceEditRequest{Semantics: s},
			},
		}
	}
	chainMetadataOp := func(s pb.EditSemantics) *pb.DataAccessRequest {
		return &pb.DataAccessRequest{
			Operation: &pb.DataAccessRequest_ChainMetadataEditRequest{
				ChainMetadataEditRequest: &pb.ChainMetadataEditRequest{Semantics: s},
			},
		}
	}
	contractMetadataOp := func(s pb.EditSemantics) *pb.DataAccessRequest {
		return &pb.DataAccessRequest{
			Operation: &pb.DataAccessRequest_ContractMetadataEditRequest{
				ContractMetadataEditRequest: &pb.ContractMetadataEditRequest{Semantics: s},
			},
		}
	}
	envMetadataOp := func(s pb.EditSemantics) *pb.DataAccessRequest {
		return &pb.DataAccessRequest{
			Operation: &pb.DataAccessRequest_EnvironmentMetadataEditRequest{
				EnvironmentMetadataEditRequest: &pb.EnvironmentMetadataEditRequest{Semantics: s},
			},
		}
	}

	tests := []struct {
		name        string
		req         *pb.DataAccessRequest
		expected    string
		expectedErr string
	}{
		// address ref
		{name: "add address ref", req: addressRefOp(pb.EditSemantics_SEMANTICS_INSERT), expected: "add address ref"},
		{name: "upsert address ref", req: addressRefOp(pb.EditSemantics_SEMANTICS_UPSERT), expected: "upsert address ref"},
		{name: "update address ref", req: addressRefOp(pb.EditSemantics_SEMANTICS_UPDATE), expected: "update address ref"},
		{name: "delete address ref", req: addressRefOp(pb.EditSemantics_SEMANTICS_DELETE), expected: "delete address ref"},
		// chain metadata
		{name: "add chain metadata", req: chainMetadataOp(pb.EditSemantics_SEMANTICS_INSERT), expected: "add chain metadata"},
		{name: "upsert chain metadata", req: chainMetadataOp(pb.EditSemantics_SEMANTICS_UPSERT), expected: "upsert chain metadata"},
		{name: "update chain metadata", req: chainMetadataOp(pb.EditSemantics_SEMANTICS_UPDATE), expected: "update chain metadata"},
		{name: "delete chain metadata", req: chainMetadataOp(pb.EditSemantics_SEMANTICS_DELETE), expected: "delete chain metadata"},
		// contract metadata
		{name: "add contract metadata", req: contractMetadataOp(pb.EditSemantics_SEMANTICS_INSERT), expected: "add contract metadata"},
		{name: "upsert contract metadata", req: contractMetadataOp(pb.EditSemantics_SEMANTICS_UPSERT), expected: "upsert contract metadata"},
		{name: "update contract metadata", req: contractMetadataOp(pb.EditSemantics_SEMANTICS_UPDATE), expected: "update contract metadata"},
		{name: "delete contract metadata", req: contractMetadataOp(pb.EditSemantics_SEMANTICS_DELETE), expected: "delete contract metadata"},
		// env metadata
		{name: "add env metadata", req: envMetadataOp(pb.EditSemantics_SEMANTICS_INSERT), expected: "add env metadata"},
		{name: "upsert env metadata", req: envMetadataOp(pb.EditSemantics_SEMANTICS_UPSERT), expected: "upsert env metadata"},
		{name: "update env metadata", req: envMetadataOp(pb.EditSemantics_SEMANTICS_UPDATE), expected: "update env metadata"},
		// edge cases
		{name: "unknown operation type", req: &pb.DataAccessRequest{}, expectedErr: "unknown operation type"},
		{name: "unknown semantics", req: &pb.DataAccessRequest{
			Operation: &pb.DataAccessRequest_AddressReferenceEditRequest{
				AddressReferenceEditRequest: &pb.AddressReferenceEditRequest{
					Semantics: 100,
				},
			},
		}, expectedErr: "unknown semantics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			op, err := getOpName(tt.req)

			if tt.expectedErr != "" {
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, op)
			}
		})
	}
}
