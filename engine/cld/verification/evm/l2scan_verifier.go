package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newL2ScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnL2Scan(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the L2Scan API", cfg.Chain.EvmChainID)
	}

	return &l2scanVerifier{cfg: cfg}, nil
}

type l2scanVerifier struct {
	cfg VerifierConfig
}

func (v *l2scanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *l2scanVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *l2scanVerifier) Verify(context.Context) error {
	return errors.New("L2Scan verifier not yet implemented in framework")
}
