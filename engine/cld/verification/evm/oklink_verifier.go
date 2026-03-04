package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newOkLinkVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnOkLink(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the OKLink API", cfg.Chain.EvmChainID)
	}

	return &oklinkVerifier{cfg: cfg}, nil
}

type oklinkVerifier struct {
	cfg VerifierConfig
}

func (v *oklinkVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *oklinkVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *oklinkVerifier) Verify(context.Context) error {
	return errors.New("OKLink verifier not yet implemented in framework")
}
