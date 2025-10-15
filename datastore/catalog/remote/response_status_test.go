package remote

import (
	"testing"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResponseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus *pb.ResponseStatus
		expectedCode   codes.Code
		expectedMsg    string
	}{
		{
			name:           "nil_response_status",
			responseStatus: nil,
			expectedCode:   codes.Internal,
			expectedMsg:    "missing response status",
		},
		{
			name: "success_code_ok",
			responseStatus: &pb.ResponseStatus{
				Code:    int32(codes.OK),
				Message: "Operation completed successfully",
			},
		},
		{
			name: "with_error",
			responseStatus: &pb.ResponseStatus{
				Code:    int32(codes.NotFound),
				Message: "No records found.",
			},
			expectedCode: codes.NotFound,
			expectedMsg:  "No records found.",
		},
		{
			name: "empty_message",
			responseStatus: &pb.ResponseStatus{
				Code:    int32(codes.NotFound),
				Message: "",
			},
			expectedCode: codes.NotFound,
			expectedMsg:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := parseResponseStatus(tt.responseStatus)

			if tt.expectedCode != codes.OK {
				require.Error(t, err)

				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.expectedCode, st.Code())
				assert.Equal(t, tt.expectedMsg, st.Message())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
