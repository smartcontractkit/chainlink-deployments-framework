package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/proto"
)

// retryPolicy derives the service name from the generated gRPC ServiceDesc so it
// always matches the proto definition, even if the proto package is renamed.
// Note: gRPC retry policies only apply to unary RPCs; the bidirectional streaming
// DataAccess RPC is never retried by gRPC.
var retryPolicy = fmt.Sprintf(`{
	"methodConfig": [{
		"name": [{"service": %q}],
		"retryPolicy": {
			"maxAttempts": 5,
			"initialBackoff": "0.1s",
			"maxBackoff": "1s",
			"backoffMultiplier": 2,
			"retryableStatusCodes": [
				"UNAVAILABLE",
				"DEADLINE_EXCEEDED",
				"INTERNAL",
				"RESOURCE_EXHAUSTED"
			]
		}
	}]
}`, pb.Datastore_ServiceDesc.ServiceName)

type CatalogClient struct {
	protoClient pb.DatastoreClient
	// ctx is cached here, because we need the context that created the client, not the current
	// call stack context. This is different than the go norm, but because we need a long-lived
	// comms session to the gRPC server, anything cancelling that context (such as a test ending)
	// would result in a dangling context.
	//
	// Another way to express this, is that this is analogous to the "request-scoped" exception to
	// passing context down the call-stack.
	//
	//nolint:containedctx
	ctx           context.Context
	conn          *grpc.ClientConn
	hmacConfig    *HMACAuthConfig
	kmsClient     kmsClient
	kmsClientOnce sync.Once
	kmsClientErr  error

	// mu guards the stream state below. Each request/response pair runs under it, so concurrent
	// callers cannot receive each other's responses.
	mu           sync.Mutex
	stream       grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse]
	cancelStream context.CancelFunc
	// inTransaction is set while a transaction is open on the stream. The server scopes
	// transactions to a stream, so the stream is kept open until the transaction ends.
	inTransaction bool
}

// DataAccess returns the client's current stream, opening one if there is none. When HMAC
// authentication is enabled, req is the message signed to authenticate the stream.
func (c *CatalogClient) DataAccess(req proto.Message) (grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse], error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.openStreamLocked(req)
}

func (c *CatalogClient) openStreamLocked(req proto.Message) (grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse], error) {
	if c.stream != nil {
		return c.stream, nil
	}

	// Each stream gets its own context, so closing it releases its resources without affecting
	// the client's context.
	ctx, cancel := context.WithCancel(c.ctx)
	if c.hmacConfig != nil {
		var err error
		ctx, err = c.prepareHMACContext(ctx, req)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to prepare HMAC context: %w", err)
		}
	}

	stream, err := c.protoClient.DataAccess(ctx)
	if err != nil {
		cancel()
		return nil, err
	}
	c.stream = stream
	c.cancelStream = cancel

	return stream, nil
}

// roundTrip sends req and returns the server's response. Outside a transaction, each request
// gets its own stream, closed once the response is received, so no stream is left idle. A
// transaction keeps its stream from the begin request until the commit or rollback.
func (c *CatalogClient) roundTrip(req *pb.DataAccessRequest) (*pb.DataAccessResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	stream, err := c.openStreamLocked(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create data access stream: %w", err)
	}

	if err = stream.Send(req); err != nil {
		c.discardStreamLocked()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		c.discardStreamLocked()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("unexpected end of stream")
		}

		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	switch req.Operation.(type) {
	case *pb.DataAccessRequest_BeginTransactionRequest:
		// A rejected begin (e.g. a nested one) leaves any transaction already open untouched.
		c.inTransaction = c.inTransaction || parseResponseStatus(resp.Status) == nil
	case *pb.DataAccessRequest_CommitTransactionRequest, *pb.DataAccessRequest_RollbackTransactionRequest:
		c.inTransaction = false
	}
	if !c.inTransaction {
		// The response has been received, so a failure to close cleanly does not affect it.
		_ = c.closeStreamLocked()
	}

	return resp, nil
}

// CloseStream closes the current stream, if any. The client stays usable: the next request
// opens a new stream.
func (c *CatalogClient) CloseStream() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeStreamLocked()
}

// closeStreamLocked half-closes the stream and waits for the server to end it, so the server
// sees a clean end of stream rather than a cancellation, then releases the stream's context.
func (c *CatalogClient) closeStreamLocked() error {
	if c.stream == nil {
		return nil
	}
	defer c.discardStreamLocked()

	if err := c.stream.CloseSend(); err != nil {
		return err
	}
	if _, err := c.stream.Recv(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

// discardStreamLocked drops the current stream without waiting for the server, cancelling its
// context. Any open transaction is lost with it; the server rolls it back.
func (c *CatalogClient) discardStreamLocked() {
	if c.cancelStream != nil {
		c.cancelStream()
	}
	c.stream = nil
	c.cancelStream = nil
	c.inTransaction = false
}

// Close closes the underlying gRPC connection.
func (c *CatalogClient) Close() error {
	c.mu.Lock()
	open := c.stream != nil
	c.mu.Unlock()
	if open {
		return errors.New("stream is not closed")
	}

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

type CatalogConfig struct {
	GRPC       string
	Creds      credentials.TransportCredentials
	HMACConfig *HMACAuthConfig
}

// NewCatalogClient creates a new CatalogClient with the provided configuration.
//
// Example usage:
//
//	cfg := CatalogConfig{
//		GRPC:  "op-catalog.example.com:443",
//		Creds: credentials.NewTLS(&tls.Config{}),
//		HMACConfig: &HMACAuthConfig{
//			KeyID:     "kms-key-id",
//			KeyRegion: "us-west-2",
//			Authority: "op-catalog.example.com",
//		},
//	}
//	client, err := NewCatalogClient(ctx, cfg)
func NewCatalogClient(ctx context.Context, cfg CatalogConfig) (*CatalogClient, error) {
	// Create connection with the configured options.
	conn, err := newCatalogConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect Catalog service. Err: %w", err)
	}

	client := CatalogClient{
		ctx:         ctx,
		hmacConfig:  cfg.HMACConfig,
		conn:        conn,
		protoClient: pb.NewDatastoreClient(conn),
	}

	return &client, nil
}

// newCatalogConnection creates a new gRPC connection to the Catalog service.
func newCatalogConnection(cfg CatalogConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if cfg.Creds != nil {
		opts = append(opts, grpc.WithTransportCredentials(cfg.Creds))
	}

	//	Force authority header to be set to match what's used in the HMAC signature, this ensures the server verifies against the
	//	same authority we signed with. If not set explicitly, the authority is derived from the grpc URL, which may not match the
	//	authority used in the HMAC signature since gRPC clients take some liberties with the authority header like removing the port
	//	E.g. if it is default 443, the authority header will be "grpc.example.com" instead of "grpc.example.com:443"
	//	see: https://github.com/grpc/grpc-go/blob/7472d578b15f718cbe8ca0f5f5a3713093c47b03/internal/transport/http2_client.go#L653
	//	see: https://github.com/grpc/grpc-go/blob/7472d578b15f718cbe8ca0f5f5a3713093c47b03/internal/transport/http2_client.go#L533
	if cfg.HMACConfig != nil {
		opts = append(opts, grpc.WithAuthority(cfg.HMACConfig.Authority))
	}

	// Keepalive for long-lived bidirectional streams
	// Ping every 20 seconds, wait up to 10 seconds for a response
	opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
		Time:                20 * time.Second,
		Timeout:             10 * time.Second,
		PermitWithoutStream: true,
	}))

	opts = append(opts, grpc.WithDefaultServiceConfig(retryPolicy))

	conn, err := grpc.NewClient(cfg.GRPC, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
