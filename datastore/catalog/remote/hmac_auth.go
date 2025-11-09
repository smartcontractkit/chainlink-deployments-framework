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
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

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

	return c.prepareHMACContextWithClient(ctx, req, kmsClient)
}

// prepareHMACContextWithClient prepares the context with HMAC authentication metadata using the provided KMS client.
// This method is extracted for testability.
func (c *CatalogClient) prepareHMACContextWithClient(ctx context.Context, req proto.Message, client kmsClient) (context.Context, error) {
	// Create HMAC helper
	hmacHelper := &kmsHMACClientHelper{
		kmsClient: client,
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

// kmsClient defines the interface for KMS operations needed for HMAC
type kmsClient interface {
	GenerateMac(ctx context.Context, params *kms.GenerateMacInput, optFns ...func(*kms.Options)) (*kms.GenerateMacOutput, error)
}

// kmsHMACClientHelper helps clients generate HMAC signatures using AWS KMS.
type kmsHMACClientHelper struct {
	kmsClient kmsClient
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
