package kms

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	kmsv2 "github.com/aws/aws-sdk-go-v2/service/kms"
	kmstypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// Ed25519Client is the subset of the aws-sdk-go-v2 KMS client used to sign with
// an Ed25519 key.
type Ed25519Client interface {
	GetPublicKey(ctx context.Context, in *kmsv2.GetPublicKeyInput, optFns ...func(*kmsv2.Options)) (*kmsv2.GetPublicKeyOutput, error)
	Sign(ctx context.Context, in *kmsv2.SignInput, optFns ...func(*kmsv2.Options)) (*kmsv2.SignOutput, error)
}

var _ Ed25519Client = (*kmsv2.Client)(nil)

// NewEd25519Client constructs an aws-sdk-go-v2 KMS client for the given region.
func NewEd25519Client(ctx context.Context, keyRegion string) (*kmsv2.Client, error) {
	if keyRegion == "" {
		return nil, errors.New("KMS key region is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(keyRegion))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return kmsv2.NewFromConfig(awsCfg), nil
}

// Ed25519KMSSigner signs arbitrary messages with an Ed25519
// key held in AWS KMS.
type Ed25519KMSSigner struct {
	client Ed25519Client
	keyID  string
	// Cached Ed25519 public key fetched from KMS.
	pubKey ed25519.PublicKey
}

// NewEd25519KMSSigner fetches and validates the public key.
func NewEd25519KMSSigner(ctx context.Context, keyID string, client Ed25519Client) (*Ed25519KMSSigner, error) {
	if keyID == "" {
		return nil, errors.New("KMS key ID is required")
	}
	if client == nil {
		return nil, errors.New("KMS client is required")
	}

	s := &Ed25519KMSSigner{client: client, keyID: keyID}
	if _, err := s.GetPublicKey(ctx); err != nil {
		return nil, err
	}

	return s, nil
}

// GetPublicKey fetches, validates, and caches the Ed25519 public key.
func (s *Ed25519KMSSigner) GetPublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	if s.pubKey != nil {
		// Return a copy: ed25519.PublicKey is a []byte, so handing out the cached
		// slice would let callers mutate the signer's key.
		return slices.Clone(s.pubKey), nil
	}

	out, err := s.client.GetPublicKey(ctx, &kmsv2.GetPublicKeyInput{
		KeyId: aws.String(s.keyID),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get public key from KMS for KeyId=%s: %w", s.keyID, err)
	}

	if out.KeySpec != kmstypes.KeySpecEccNistEdwards25519 {
		return nil, fmt.Errorf(
			"KMS key %s has key spec %s, want %s",
			s.keyID, out.KeySpec, kmstypes.KeySpecEccNistEdwards25519,
		)
	}

	parsed, err := x509.ParsePKIXPublicKey(out.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot parse public key for KeyId=%s: %w", s.keyID, err)
	}

	pub, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key for KeyId=%s is %T, want ed25519.PublicKey", s.keyID, parsed)
	}

	s.pubKey = pub

	return slices.Clone(s.pubKey), nil
}

// Sign signs message with the KMS Ed25519 key and returns the flat 64-byte
// signature.
func (s *Ed25519KMSSigner) Sign(ctx context.Context, message []byte) ([]byte, error) {
	out, err := s.client.Sign(ctx, &kmsv2.SignInput{
		KeyId:            aws.String(s.keyID),
		Message:          message,
		MessageType:      kmstypes.MessageTypeRaw,
		SigningAlgorithm: kmstypes.SigningAlgorithmSpecEd25519Sha512,
	})
	if err != nil {
		return nil, fmt.Errorf("call to kms.Sign() failed: %w", err)
	}

	if len(out.Signature) != ed25519.SignatureSize {
		return nil, fmt.Errorf("unexpected signature length %d, want %d", len(out.Signature), ed25519.SignatureSize)
	}

	return out.Signature, nil
}
