package jd

import (
	"context"
	"errors"
	"fmt"

	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// JDConfig is the configuration for the Job Distributor client.
type JDConfig struct {
	GRPC  string
	WSRPC string
	Creds credentials.TransportCredentials
	Auth  oauth2.TokenSource
}

// JobDistributor is the client for the Job Distributor service.
type JobDistributor struct {
	nodev1.NodeServiceClient
	jobv1.JobServiceClient
	csav1.CSAServiceClient

	WSRPC string
}

// NewJDClient creates a new Job Distributor client
func NewJDClient(cfg JDConfig) (*JobDistributor, error) {
	conn, err := newJDConnection(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect Job Distributor service. Err: %w", err)
	}
	jd := &JobDistributor{
		WSRPC:             cfg.WSRPC,
		NodeServiceClient: nodev1.NewNodeServiceClient(conn),
		JobServiceClient:  jobv1.NewJobServiceClient(conn),
		CSAServiceClient:  csav1.NewCSAServiceClient(conn),
	}

	return jd, err
}

// GetCSAPublicKey returns the public key for the CSA service
func (jd *JobDistributor) GetCSAPublicKey(ctx context.Context) (string, error) {
	keypairs, err := jd.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
	if err != nil {
		return "", err
	}
	if keypairs == nil || len(keypairs.Keypairs) == 0 {
		return "", errors.New("no keypairs found")
	}
	csakey := keypairs.Keypairs[0].PublicKey

	return csakey, nil
}

// ProposeJob proposes jobs through the jobService and accepts the proposed job on selected node based on ProposeJobRequest.NodeId
func (jd *JobDistributor) ProposeJob(ctx context.Context, in *jobv1.ProposeJobRequest, opts ...grpc.CallOption) (*jobv1.ProposeJobResponse, error) {
	res, err := jd.JobServiceClient.ProposeJob(ctx, in, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to propose job. err: %w", err)
	}
	if res.Proposal == nil {
		return nil, errors.New("failed to propose job. err: proposal is nil")
	}

	return res, nil
}

// newJDConnection creates a new connection to the Job Distributor service.
func newJDConnection(cfg JDConfig) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{}
	interceptors := []grpc.UnaryClientInterceptor{}

	if cfg.Creds != nil {
		opts = append(opts, grpc.WithTransportCredentials(cfg.Creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if cfg.Auth != nil {
		interceptors = append(interceptors, authTokenInterceptor(cfg.Auth))
	}

	if len(interceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(interceptors...))
	}

	conn, err := grpc.NewClient(cfg.GRPC, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect Job Distributor service. Err: %w", err)
	}

	return conn, nil
}
