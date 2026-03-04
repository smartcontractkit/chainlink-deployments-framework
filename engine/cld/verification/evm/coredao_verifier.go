package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newCoreDAOVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnCoreDAO(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the CoreDAO API", cfg.Chain.EvmChainID)
	}

	return &coredaoVerifier{cfg: cfg}, nil
}

type coredaoVerifier struct {
	cfg VerifierConfig
}

func (v *coredaoVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *coredaoVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *coredaoVerifier) Verify(context.Context) error {
	return errors.New("CoreDAO verifier not yet implemented in framework")
}
