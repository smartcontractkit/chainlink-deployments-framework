package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
)

const (
	streamEndTimeout = 5 * time.Second
	streamEndTick    = 10 * time.Millisecond
)

// streamEnd records how a stream ended on the fake server.
type streamEnd struct {
	messages int
	eof      bool // true if the client half-closed the stream, false if it was cancelled
}

// fakeDatastoreServer answers every request on a stream and records how each stream ended.
// Find requests are answered with their qualifier filter echoed in the status message, so tests
// can match responses to requests. Begin requests fail when failBegin is set or, as on the real
// server, when a transaction is already open on the stream. A request whose qualifier is "drop"
// makes the server end the stream without answering.
type fakeDatastoreServer struct {
	pb.UnimplementedDatastoreServer

	failBegin bool

	mu   sync.Mutex
	ends []streamEnd
}

func (f *fakeDatastoreServer) DataAccess(stream grpc.BidiStreamingServer[pb.DataAccessRequest, pb.DataAccessResponse]) error {
	end := streamEnd{}
	inTx := false
	defer func() {
		f.mu.Lock()
		f.ends = append(f.ends, end)
		f.mu.Unlock()
	}()

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			end.eof = true
			return nil
		}
		if err != nil {
			return err
		}
		end.messages++

		resp := &pb.DataAccessResponse{Status: &pb.ResponseStatus{}}
		switch op := req.Operation.(type) {
		case *pb.DataAccessRequest_BeginTransactionRequest:
			if f.failBegin || inTx {
				resp.Status = &pb.ResponseStatus{Code: int32(codes.FailedPrecondition), Message: "begin failed"}
			} else {
				inTx = true
			}
		case *pb.DataAccessRequest_CommitTransactionRequest, *pb.DataAccessRequest_RollbackTransactionRequest:
			inTx = false
		case *pb.DataAccessRequest_AddressReferenceFindRequest:
			qualifier := op.AddressReferenceFindRequest.GetKeyFilter().GetQualifier().GetValue()
			if qualifier == "drop" {
				return status.Error(codes.Unavailable, "dropped")
			}
			resp.Status.Message = qualifier
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// streamEnds returns the ends recorded so far. A stream is recorded once its handler returns,
// which can be just after the client has moved on, so callers wait for the count they expect.
func (f *fakeDatastoreServer) streamEnds(t *testing.T, want int) []streamEnd {
	t.Helper()

	var ends []streamEnd
	require.Eventually(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		ends = append([]streamEnd(nil), f.ends...)

		return len(ends) >= want
	}, streamEndTimeout, streamEndTick)

	return ends
}

func newFakeCatalogClient(t *testing.T, srv *fakeDatastoreServer) *CatalogClient {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pb.RegisterDatastoreServer(server, srv)
	go func() { _ = server.Serve(lis) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &CatalogClient{ctx: t.Context(), conn: conn, protoClient: pb.NewDatastoreClient(conn)}
}

func findRequest(qualifier string) *pb.DataAccessRequest {
	return &pb.DataAccessRequest{
		Operation: &pb.DataAccessRequest_AddressReferenceFindRequest{
			AddressReferenceFindRequest: &pb.AddressReferenceFindRequest{
				KeyFilter: &pb.AddressReferenceKeyFilter{Qualifier: wrapperspb.String(qualifier)},
			},
		},
	}
}

var (
	beginRequest = &pb.DataAccessRequest{Operation: &pb.DataAccessRequest_BeginTransactionRequest{
		BeginTransactionRequest: &pb.BeginTransactionRequest{},
	}}
	commitRequest = &pb.DataAccessRequest{Operation: &pb.DataAccessRequest_CommitTransactionRequest{
		CommitTransactionRequest: &pb.CommitTransactionRequest{},
	}}
	rollbackRequest = &pb.DataAccessRequest{Operation: &pb.DataAccessRequest_RollbackTransactionRequest{
		RollbackTransactionRequest: &pb.RollbackTransactionRequest{},
	}}
)

func TestCatalogClient_roundTrip_StreamPerRequest(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{}
	client := newFakeCatalogClient(t, srv)

	for i := range 3 {
		resp, err := client.roundTrip(findRequest(strconv.Itoa(i)))
		require.NoError(t, err)
		assert.Equal(t, strconv.Itoa(i), resp.Status.Message)
	}

	// Each request got its own stream, closed cleanly once answered.
	assert.Equal(t, []streamEnd{{messages: 1, eof: true}, {messages: 1, eof: true}, {messages: 1, eof: true}},
		srv.streamEnds(t, 3))
	assert.Nil(t, client.stream)
	require.NoError(t, client.Close())
}

func TestCatalogClient_roundTrip_TransactionKeepsStream(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		end  *pb.DataAccessRequest
	}{
		{name: "commit", end: commitRequest},
		{name: "rollback", end: rollbackRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := &fakeDatastoreServer{}
			client := newFakeCatalogClient(t, srv)

			_, err := client.roundTrip(beginRequest)
			require.NoError(t, err)
			_, err = client.roundTrip(findRequest("a"))
			require.NoError(t, err)
			_, err = client.roundTrip(findRequest("b"))
			require.NoError(t, err)
			assert.NotNil(t, client.stream, "stream must stay open during the transaction")

			_, err = client.roundTrip(tt.end)
			require.NoError(t, err)

			// Begin, both finds and the end of the transaction all used one stream.
			assert.Equal(t, []streamEnd{{messages: 4, eof: true}}, srv.streamEnds(t, 1))
			assert.Nil(t, client.stream)
		})
	}
}

func TestCatalogClient_roundTrip_FailedBeginClosesStream(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{failBegin: true}
	client := newFakeCatalogClient(t, srv)

	resp, err := client.roundTrip(beginRequest)
	require.NoError(t, err)
	require.Error(t, parseResponseStatus(resp.Status))

	assert.Equal(t, []streamEnd{{messages: 1, eof: true}}, srv.streamEnds(t, 1))
	assert.False(t, client.inTransaction)
	assert.Nil(t, client.stream)
}

func TestCatalogClient_roundTrip_RejectedNestedBeginKeepsTransaction(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{}
	client := newFakeCatalogClient(t, srv)

	_, err := client.roundTrip(beginRequest)
	require.NoError(t, err)

	// The server rejects the nested begin; the open transaction and its stream must survive.
	resp, err := client.roundTrip(beginRequest)
	require.NoError(t, err)
	require.Error(t, parseResponseStatus(resp.Status))
	assert.True(t, client.inTransaction)
	assert.NotNil(t, client.stream)

	_, err = client.roundTrip(commitRequest)
	require.NoError(t, err)
	assert.Equal(t, []streamEnd{{messages: 3, eof: true}}, srv.streamEnds(t, 1))
}

func TestCatalogClient_roundTrip_RecoversFromBrokenStream(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{}
	client := newFakeCatalogClient(t, srv)

	_, err := client.roundTrip(beginRequest)
	require.NoError(t, err)

	// The server ends the stream mid-transaction.
	_, err = client.roundTrip(findRequest("drop"))
	require.ErrorContains(t, err, "failed to receive response")
	assert.False(t, client.inTransaction, "a broken stream ends the transaction")
	assert.Nil(t, client.stream)

	// The next request opens a fresh stream.
	resp, err := client.roundTrip(findRequest("after"))
	require.NoError(t, err)
	assert.Equal(t, "after", resp.Status.Message)
	assert.Len(t, srv.streamEnds(t, 2), 2)
}

func TestCatalogClient_CloseStream_ClientStaysUsable(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{}
	client := newFakeCatalogClient(t, srv)

	_, err := client.DataAccess(&pb.DataAccessRequest{})
	require.NoError(t, err)
	require.NoError(t, client.CloseStream())
	require.NoError(t, client.CloseStream(), "closing twice is a no-op")

	resp, err := client.roundTrip(findRequest("again"))
	require.NoError(t, err)
	assert.Equal(t, "again", resp.Status.Message)
	assert.Equal(t, []streamEnd{{messages: 0, eof: true}, {messages: 1, eof: true}}, srv.streamEnds(t, 2))
}

func TestCatalogClient_roundTrip_ConcurrentCallers(t *testing.T) {
	t.Parallel()

	srv := &fakeDatastoreServer{}
	client := newFakeCatalogClient(t, srv)

	const callers = 20
	var wg sync.WaitGroup
	errs := make([]error, callers)
	for i := range callers {
		wg.Go(func() {
			want := strconv.Itoa(i)
			resp, err := client.roundTrip(findRequest(want))
			if err == nil && resp.Status.Message != want {
				err = fmt.Errorf("caller %s got the response for %s", want, resp.Status.Message)
			}
			errs[i] = err
		})
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}
}
