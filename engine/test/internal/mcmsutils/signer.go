package mcmsutils

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

// Signer provides functionality for signing MCMS and Timelock proposals.
// It manages the environment configuration and handles the conversion of timelock
// proposals to MCMS proposals before signing.
type Signer struct {
	// isEVMSim is a flag indicating whether to use simulated EVM backends.
	isEVMSim bool
}

// NewSigner creates a new MCMS signer with the provided environment configuration.
//
// The signer is configured to use simulated EVM backends by default, which affects
// how encoders are generated for signing EVM proposals.
func NewSigner() (*Signer, error) {
	return &Signer{
		isEVMSim: true, // This is always true for until we can find a way to allow the user to specify the type of EVM backend they are using.
	}, nil
}

// SignMCMS signs an MCMS proposal using the provided private key and appends the generated
// signature to the proposal's signatures list.
func (s *Signer) SignMCMS(
	ctx context.Context,
	proposal *mcmslib.Proposal,
	privateKey *ecdsa.PrivateKey,
) error {
	// Validate the proposal to ensure it is valid. This ensures that all chain metadata is present.
	if err := proposal.Validate(); err != nil {
		return fmt.Errorf("failed to validate MCMS proposal: %w", err)
	}

	proposal.UseSimulatedBackend(s.isEVMSim)

	sig, err := signProposal(*proposal, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign MCMS proposal: %w", err)
	}

	proposal.AppendSignature(sig)

	return nil
}

// SignTimelock signs a timelock proposal by first converting it to an MCMS proposal, signing it,
// and appending the signature back to the original timelock proposal.
func (s *Signer) SignTimelock(
	ctx context.Context,
	timelockProposal *mcmslib.TimelockProposal,
	privateKey *ecdsa.PrivateKey,
) error {
	// Validate the proposal to ensure it is valid. This ensures that all chain metadata is present.
	if err := timelockProposal.Validate(); err != nil {
		return fmt.Errorf("failed to validate MCMS proposal: %w", err)
	}

	p, err := convertTimelock(ctx, *timelockProposal)
	if err != nil {
		return fmt.Errorf("failed to convert timelock proposal to MCMS proposal: %w", err)
	}

	p.UseSimulatedBackend(s.isEVMSim)

	sig, err := signProposal(*p, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign converted MCMS proposal: %w", err)
	}

	// Append the signature to the Timelock proposal
	timelockProposal.AppendSignature(sig)

	return nil
}

// signProposal is a helper function that generates a cryptographic signature for an MCMS proposal.
// It creates a signing message from the proposal and uses the provided private key to sign it.
func signProposal(
	proposal mcmslib.Proposal, privateKey *ecdsa.PrivateKey,
) (mcmstypes.Signature, error) {
	signer := mcmslib.NewPrivateKeySigner(privateKey)

	// Get the signing hash
	payload, err := proposal.SigningMessage()
	if err != nil {
		return mcmstypes.Signature{}, err
	}

	// Sign the payload
	sig, err := signer.Sign(payload.Bytes())
	if err != nil {
		return mcmstypes.Signature{}, err
	}

	return mcmstypes.NewSignatureFromBytes(sig)
}
