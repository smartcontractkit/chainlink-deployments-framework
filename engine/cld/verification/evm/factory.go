package evm

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	chainsel "github.com/smartcontractkit/chain-selectors"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// VerifierConfig holds the parameters needed to create a verifier.
type VerifierConfig struct {
	Chain        chainsel.Chain
	Network      cfgnet.Network
	Address      string
	Metadata     SolidityContractMetadata
	ContractType string
	Version      string
	PollInterval time.Duration
	Logger       logger.Logger
	// HTTPClient is optional; when nil, verifiers use http.DefaultClient. Set for testing.
	HTTPClient *http.Client
}

// NewVerifier creates a Verifiable for the given strategy and config.
func NewVerifier(strategy VerificationStrategy, cfg VerifierConfig) (verification.Verifiable, error) {
	switch strategy {
	case StrategyUnknown:
		return nil, errors.New("no verifier for unknown strategy")
	case StrategyEtherscan:
		apiKey := cfg.Network.BlockExplorer.APIKey
		if apiKey == "" {
			return nil, fmt.Errorf("etherscan API key not configured for chain %s", cfg.Chain.Name)
		}

		return NewEtherscanV2ContractVerifier(
			cfg.Chain, apiKey, cfg.Address, cfg.Metadata,
			cfg.ContractType, cfg.Version, cfg.PollInterval, cfg.Logger,
			cfg.HTTPClient,
		)
	case StrategyRoutescan:
		return newRouteScanVerifier(cfg)
	case StrategyBlockscout:
		return newBlockscoutVerifier(cfg)
	case StrategySourcify:
		return newSourcifyVerifier(cfg)
	case StrategyOkLink:
		return newOkLinkVerifier(cfg)
	case StrategyBtrScan:
		return newBtrScanVerifier(cfg)
	case StrategyCoreDAO:
		return newCoreDAOVerifier(cfg)
	case StrategyL2Scan:
		return newL2ScanVerifier(cfg)
	case StrategySocialScan:
		return newSocialScanVerifier(cfg)
	default:
		return nil, fmt.Errorf("no verifier for strategy %d", strategy)
	}
}
