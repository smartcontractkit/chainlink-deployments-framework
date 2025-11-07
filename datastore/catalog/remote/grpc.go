package remote

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	pb "github.com/smartcontractkit/chainlink-protos/op-catalog/v1/datastore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
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
	ctx          context.Context
	cachedStream grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse]
	hmacConfig   *HMACAuthConfig
}

func (c *CatalogClient) DataAccess(req proto.Message) (grpc.BidiStreamingClient[pb.DataAccessRequest, pb.DataAccessResponse], error) {
	if c.cachedStream == nil {
		// Apply HMAC signature if enabled
		ctx := c.ctx
		if c.hmacConfig != nil {
			var err error
			ctx, err = c.prepareHMACContext(c.ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to prepare HMAC context: %w", err)
			}
		}

		stream, err := c.protoClient.DataAccess(ctx)
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

const (
	// dataAccessMethod is the full gRPC method name for DataAccess
	dataAccessMethod = "/op_catalog.v1.datastore.Datastore/DataAccess"
)

// HMACAuthConfig holds HMAC authentication configuration.
type HMACAuthConfig struct {
	KeyID     string
	KeyRegion string
	Authority string // The gRPC authority (hostname without port) used for HMAC signing
}

// prepareHMACContext prepares the context with HMAC authentication metadata.
// It loads AWS KMS configuration, creates a KMS client, generates an HMAC signature,
// and attaches it to the outgoing gRPC metadata.
func (c *CatalogClient) prepareHMACContext(ctx context.Context, req proto.Message) (context.Context, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.hmacConfig.KeyRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create KMS client
	kmsClient := kms.NewFromConfig(cfg)

	// Create HMAC helper
	hmacHelper := &kmsHMACClientHelper{
		kmsClient: kmsClient,
		keyID:     c.hmacConfig.KeyID,
	}

	// Marshal the message to bytes for HMAC
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message for HMAC: %w", err)
	}

	// Generate HMAC signature and timestamp
	signature, timestamp, err := hmacHelper.generateHMACSignature(ctx, dataAccessMethod, c.hmacConfig.Authority, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to generate HMAC signature: %w", err)
	}

	// Add HMAC authentication to gRPC metadata
	md := metadata.Pairs(
		"x-hmac-signature", signature,
		"x-hmac-timestamp", timestamp,
	)

	// Merge with existing metadata if present
	if existingMd, ok := metadata.FromOutgoingContext(ctx); ok {
		md = metadata.Join(existingMd, md)
	}

	return metadata.NewOutgoingContext(ctx, md), nil
}

// kmsHMACClientHelper helps clients generate HMAC signatures using AWS KMS.
type kmsHMACClientHelper struct {
	kmsClient *kms.Client
	keyID     string
}

// generateHMACSignature generates an HMAC signature and timestamp for the given request.
// Returns the hex-encoded signature and Unix timestamp as strings.
// The caller is responsible for adding these to transport-specific metadata/headers.
func (h *kmsHMACClientHelper) generateHMACSignature(ctx context.Context, method string, authority string, payload []byte) (signature string, timestamp string, err error) {
	timestamp = strconv.FormatInt(time.Now().Unix(), 10)

	// Hash the payload with SHA-256 to stay within KMS message size limits (4096 bytes)
	// and to have a predictable signature length
	payloadHash := sha256.Sum256(payload)

	// Construct HMAC message using method path, authority, timestamp, and payload hash
	// Format: method\nauthority\ntimestamp\nsha256(payload)
	messagePrefix := fmt.Sprintf("%s\n%s\n%s\n", method, authority, timestamp)
	fullMessage := append([]byte(messagePrefix), payloadHash[:]...)

	// Generate MAC using KMS with HMAC_SHA_256
	generateInput := &kms.GenerateMacInput{
		KeyId:        aws.String(h.keyID),
		Message:      fullMessage,
		MacAlgorithm: types.MacAlgorithmSpecHmacSha256,
	}

	generateOutput, err := h.kmsClient.GenerateMac(ctx, generateInput)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate MAC: %w", err)
	}

	signature = hex.EncodeToString(generateOutput.Mac)

	return signature, timestamp, nil
}
