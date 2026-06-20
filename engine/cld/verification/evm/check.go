package evm

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// CheckConfig holds the configuration for checking contract verification status.
type CheckConfig struct {
	// ContractInputsProvider supplies contract metadata. Required.
	ContractInputsProvider ContractInputsProvider
	// NetworkConfig is the loaded network configuration (EVM networks only). Required.
	NetworkConfig *cfgnet.Config
	// Logger for diagnostics.
	Logger logger.Logger
	// HTTPClient is optional; when nil, verifiers use http.DefaultClient. Set for testing.
	HTTPClient *http.Client
}

// CheckVerified checks which of the given address refs are already verified on block explorers.
// Returns the refs that are NOT verified. Skips refs for chains with no verifier strategy
// (those are not included in the unverified list). Returns an error if any ref cannot be
// checked (e.g. network not in config, metadata unavailable, API error).
//
// Use this in pre-hooks to block changesets when contracts must be verified first.
func CheckVerified(ctx context.Context, refs []datastore.AddressRef, cfg CheckConfig) (unverified []datastore.AddressRef, err error) {
	if cfg.ContractInputsProvider == nil {
		return nil, errors.New("CheckConfig.ContractInputsProvider is required")
	}
	if cfg.NetworkConfig == nil {
		return nil, errors.New("CheckConfig.NetworkConfig is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = logger.Nop()
	}

	evmConfig := cfg.NetworkConfig.FilterWith(cfgnet.ChainFamilyFilter(chain_selectors.FamilyEVM))

	for _, ref := range refs {
		if ref.Version == nil {
			return nil, fmt.Errorf("address %s on chain %d: version is required", ref.Address, ref.ChainSelector)
		}

		network, err := evmConfig.NetworkBySelector(ref.ChainSelector)
		if err != nil {
			return nil, fmt.Errorf("address %s: %w", ref.Address, err)
		}

		chain, ok := chain_selectors.ChainBySelector(ref.ChainSelector)
		if !ok {
			return nil, fmt.Errorf("address %s: no chain found for selector %d", ref.Address, ref.ChainSelector)
		}

		strategy := GetVerificationStrategy(chain.EvmChainID)
		if strategy == StrategyUnknown {
			return nil, fmt.Errorf("address %s on %s: no verification strategy for chain ID %d", ref.Address, chain.Name, chain.EvmChainID)
		}

		metadata, err := cfg.ContractInputsProvider.GetInputs(ref.Type, ref.Version)
		if err != nil {
			return nil, fmt.Errorf("address %s: %w", ref.Address, err)
		}

		verifier, err := NewVerifier(strategy, VerifierConfig{
			Chain:        chain,
			Network:      network,
			Address:      ref.Address,
			Metadata:     metadata,
			ContractType: string(ref.Type),
			Version:      ref.Version.String(),
			PollInterval: 0,
			Logger:       cfg.Logger,
			HTTPClient:   cfg.HTTPClient,
		})
		if err != nil {
			return nil, fmt.Errorf("address %s on %s: %w", ref.Address, chain.Name, err)
		}

		verified, err := verifier.IsVerified(ctx)
		if err != nil {
			return nil, fmt.Errorf("address %s on %s: %w", ref.Address, chain.Name, err)
		}

		if !verified {
			unverified = append(unverified, ref)
		}
	}

	return unverified, nil
}
