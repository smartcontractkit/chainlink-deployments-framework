package catalog

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	datastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore/catalog/internal/protos"
)

type CatalogClient struct {
	GRPC string
	datastore.DeploymentsDatastoreClient
}

type CatalogConfig struct {
	GRPC  string
	Creds credentials.TransportCredentials
}

// NewCatalogClient creates a new CatalogClient with the provided configuration.
func NewCatalogClient(cfg CatalogConfig) (CatalogClient, error) {
	conn, err := newCatalogConnection(cfg)
	if err != nil {
		return CatalogClient{}, fmt.Errorf("failed to connect Catalog service. Err: %w", err)
	}
	client := CatalogClient{
		GRPC:                       cfg.GRPC,
		DeploymentsDatastoreClient: datastore.NewDeploymentsDatastoreClient(conn),
	}

	return client, err
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
