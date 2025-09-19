package mcmsutils

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	mcmslib "github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

type Signer struct {
	env fdeployment.Environment
}

func NewSigner(e fdeployment.Environment) (*Signer, error) {
	return &Signer{
		env: e,
	}, nil
}

func (s *Signer) SignTimelock(
	ctx context.Context,
	proposal *mcmslib.TimelockProposal,
	privateKey *ecdsa.PrivateKey,
	useSimulatedBackend bool, // TODO: Replace this boolean
) error {
	inspectors := make(map[mcmstypes.ChainSelector]mcmssdk.Inspector, 0)
	converters := make(map[mcmstypes.ChainSelector]mcmssdk.TimelockConverter, 0)

	// TODO: We only need to create inspectors and converters for the chains that are used in the proposal
	for _, b := range s.env.BlockChains.All() {
		sel := mcmstypes.ChainSelector(b.ChainSelector())

		inspFactory, err := GetTimelockInspectorFactory(b, proposal.Action)
		if err != nil {
			return fmt.Errorf("get timelock inspector factory for chain %d: %w", sel, err)
		}

		inspector, err := inspFactory.Make()
		if err != nil {
			return fmt.Errorf("make inspector for chain %d: %w", b.ChainSelector(), err)
		}
		inspectors[sel] = inspector

		convFactory, err := GetConverterFactory(b)
		if err != nil {
			return fmt.Errorf("get converter factory for chain %d: %w", sel, err)
		}

		converter, err := convFactory.Make()
		if err != nil {
			return fmt.Errorf("make converter for chain %d: %w", b.ChainSelector(), err)
		}
		converters[sel] = converter
	}

	p, _, err := proposal.Convert(ctx, converters)
	if err != nil {
		return fmt.Errorf("convert proposal: %w", err)
	}

	// TODO: Fix this boolean
	p.UseSimulatedBackend(useSimulatedBackend)

	signable, err := mcmslib.NewSignable(&p, inspectors)
	if err != nil {
		return fmt.Errorf("new signable for chain: %w", err)
	}

	if err = signable.ValidateConfigs(ctx); err != nil {
		return fmt.Errorf("validate configs: %w", err)
	}

	signer := mcmslib.NewPrivateKeySigner(privateKey)
	if _, err = signable.SignAndAppend(signer); err != nil {
		return fmt.Errorf("sign and append: %w", err)
	}

	// quorumMet, err := signable.ValidateSignatures(ctx)
	// if err != nil {
	// 	return fmt.Errorf("validate signatures: %w", err)
	// }
	// if !quorumMet {
	// 	return fmt.Errorf("quorum not met")
	// }

	return nil
}

func (s *Signer) SignMCMS(
	ctx context.Context,
	proposal *mcmslib.Proposal,
	privateKey *ecdsa.PrivateKey,
	useSimulatedBackend bool,
) error {
	inspectors := make(map[mcmstypes.ChainSelector]mcmssdk.Inspector, 0)

	for _, b := range s.env.BlockChains.All() {
		sel := mcmstypes.ChainSelector(b.ChainSelector())

		inspFactory, err := GetInspectorFactory(b)
		if err != nil {
			return fmt.Errorf("get inspector factory for chain %d: %w", sel, err)
		}

		inspector, err := inspFactory.Make()
		if err != nil {
			return fmt.Errorf("make inspector for chain %d: %w", b.ChainSelector(), err)
		}
		inspectors[sel] = inspector
	}

	// TODO: Fixme
	proposal.UseSimulatedBackend(useSimulatedBackend)

	signable, err := mcmslib.NewSignable(proposal, inspectors)
	if err != nil {
		return fmt.Errorf("new signable for chain: %w", err)
	}

	if err = signable.ValidateConfigs(ctx); err != nil {
		return fmt.Errorf("validate configs: %w", err)
	}

	signer := mcmslib.NewPrivateKeySigner(privateKey)
	if _, err = signable.SignAndAppend(signer); err != nil { // SignAndAppend directly adds the signature to the proposal
		return fmt.Errorf("sign and append: %w", err)
	}

	return nil
}
