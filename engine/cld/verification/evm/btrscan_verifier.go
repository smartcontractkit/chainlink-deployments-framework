package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newBtrScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnBtrScan(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the BtrScan API", cfg.Chain.EvmChainID)
	}

	return &btrscanVerifier{cfg: cfg}, nil
}

type btrscanVerifier struct {
	cfg VerifierConfig
}

func (v *btrscanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *btrscanVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *btrscanVerifier) Verify(context.Context) error {
	return errors.New("BtrScan verifier not yet implemented in framework")
}
