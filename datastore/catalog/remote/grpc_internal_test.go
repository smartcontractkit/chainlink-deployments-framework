package remote

import (
	"encoding/json"
	"testing"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRetryPolicy_ServiceNameMatchesProto ensures the retry policy's service name
// is derived from the generated gRPC ServiceDesc and cannot drift from the proto
// definition. The service config must also be valid JSON, as an unparseable config
// would fail connection creation at dial time.
func TestRetryPolicy_ServiceNameMatchesProto(t *testing.T) {
	t.Parallel()

	var config struct {
		MethodConfig []struct {
			Name []struct {
				Service string `json:"service"`
			} `json:"name"`
		} `json:"methodConfig"`
	}

	require.NoError(t, json.Unmarshal([]byte(retryPolicy), &config))
	require.NotEmpty(t, config.MethodConfig)
	require.NotEmpty(t, config.MethodConfig[0].Name)

	assert.Equal(t, pb.Datastore_ServiceDesc.ServiceName, config.MethodConfig[0].Name[0].Service)
}
