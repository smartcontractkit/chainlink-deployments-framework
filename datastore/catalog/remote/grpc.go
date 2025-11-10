package remote

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
)

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
	ctx            context.Context
	cachedStream   grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse]
	hmacConfig     *HMACAuthConfig
	streamInitOnce sync.Once
	streamInitErr  error
}

func (c *CatalogClient) DataAccess(req proto.Message) (grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse], error) {
	c.streamInitOnce.Do(func() {
		ctx := c.ctx
		if c.hmacConfig != nil {
			var err error
			ctx, err = c.prepareHMACContext(c.ctx, req)
			if err != nil {
				c.streamInitErr = fmt.Errorf("failed to prepare HMAC context: %w", err)
				return
			}
		}

		stream, err := c.protoClient.DataAccess(ctx)
		if err != nil {
			c.streamInitErr = err
			return
		}
		c.cachedStream = stream
	})

	return c.cachedStream, c.streamInitErr
}

func (c *CatalogClient) CloseStream() error {
	if c.cachedStream == nil {
		return nil
	}
	err := c.cachedStream.CloseSend()
	if err != nil {
		return err
	}
	c.cachedStream = nil

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
	client := &CatalogClient{
		ctx:        ctx,
		hmacConfig: cfg.HMACConfig,
	}

	// Create connection with the configured options
	conn, err := newCatalogConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect Catalog service. Err: %w", err)
	}

	client.protoClient = pb.NewDatastoreClient(conn)

	return client, nil
}

// newCatalogConnection creates a new gRPC connection to the Catalog service.
func newCatalogConnection(cfg CatalogConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	var interceptors []grpc.UnaryClientInterceptor

	if cfg.Creds != nil {
		opts = append(opts, grpc.WithTransportCredentials(cfg.Creds))
	}

	if cfg.HMACConfig != nil {
		// Force authority header to be set to match what's used in the HMAC signature.
		// This ensures the server verifies against the same authority we signed with.
		// see: https://github.com/grpc/grpc-go/blob/7472d578b15f718cbe8ca0f5f5a3713093c47b03/internal/transport/http2_client.go#L653
		// see: https://github.com/grpc/grpc-go/blob/7472d578b15f718cbe8ca0f5f5a3713093c47b03/internal/transport/http2_client.go#L533
		opts = append(opts, grpc.WithAuthority(cfg.HMACConfig.Authority))
	}

	if len(interceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(interceptors...))
	}

	conn, err := grpc.NewClient(cfg.GRPC, opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
