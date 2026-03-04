package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
)

func newSocialScanVerifier(cfg VerifierConfig) (verification.Verifiable, error) {
	if !IsChainSupportedOnSocialScanV2(cfg.Chain.EvmChainID) {
		return nil, fmt.Errorf("chain ID %d is not supported by the SocialScan API", cfg.Chain.EvmChainID)
	}

	return &socialscanVerifier{cfg: cfg}, nil
}

type socialscanVerifier struct {
	cfg VerifierConfig
}

func (v *socialscanVerifier) String() string {
	return fmt.Sprintf("%s %s (%s on %s)", v.cfg.ContractType, v.cfg.Version, v.cfg.Address, v.cfg.Chain.Name)
}

func (v *socialscanVerifier) IsVerified(context.Context) (bool, error) {
	return false, nil
}

func (v *socialscanVerifier) Verify(context.Context) error {
	return errors.New("SocialScan verifier not yet implemented in framework")
}
