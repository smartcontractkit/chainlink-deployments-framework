package remote

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/smartcontractkit/chainlink-protos/chainlink-catalog/v1/datastore"
)

type CatalogClient struct {
	protoClient pb.DeploymentsDatastoreClient
	// ctx is cached here, because we need the context that created the client, not the current
	// call stack context. This is different than the go norm, but because we need a long-lived
	// comms session to the gRPC server, anything cancelling that context (such as a test ending)
	// would result in a dangling context.
	//
	// Another way to express this, is that this is analogous to the "request-scoped" exception to
	// passing context down the call-stack.
	//
	//nolint:containedctx
	ctx          context.Context
	cachedStream grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse]
}

func (c *CatalogClient) DataAccess() (grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse], error) {
	if c.cachedStream == nil {
		stream, err := c.protoClient.DataAccess(c.ctx)
		if err != nil {
			return nil, err
		}
		c.cachedStream = stream
	}

	return c.cachedStream, nil
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
	GRPC  string
	Creds credentials.TransportCredentials
}

// NewCatalogClient creates a new CatalogClient with the provided configuration.
func NewCatalogClient(ctx context.Context, cfg CatalogConfig) (*CatalogClient, error) {
	conn, err := newCatalogConnection(cfg)
	if err != nil {
		return &CatalogClient{}, fmt.Errorf("failed to connect Catalog service. Err: %w", err)
	}
	client := CatalogClient{
		ctx:         ctx,
		protoClient: pb.NewDeploymentsDatastoreClient(conn),
	}

	return &client, err
}

// newCatalogConnection creates a new gRPC connection to the Catalog service.
func newCatalogConnection(cfg CatalogConfig) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	var interceptors []grpc.UnaryClientInterceptor

	if cfg.Creds != nil {
		opts = append(opts, grpc.WithTransportCredentials(cfg.Creds))
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
