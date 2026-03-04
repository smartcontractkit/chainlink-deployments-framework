package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newSourcifyVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnSourcify(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the sourcify API", cfg.Chain.EvmChainID)
	}

	return &sourcifyVerifier{cfg: cfg}, nil
}

type sourcifyVerifier struct {
	cfg VerifierConfig
}

func (v *sourcifyVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *sourcifyVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *sourcifyVerifier) Verify(context.Context) error {
	return errors.New("sourcify verifier not yet implemented in framework")
}
