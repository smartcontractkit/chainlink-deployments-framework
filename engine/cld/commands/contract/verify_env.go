package contract

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/commands/flags"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
)

var (
	verifyEnvShort = "Verify EVM contracts in an environment"

	verifyEnvLong = `Verify all EVM contracts in a given environment using the datastore.

Networks are skipped when no verification strategy applies (e.g. missing or unsupported block_explorer.type, or missing fields required for strategy selection such as URL or slug).

Some explorer types allow an empty api_key in config for local or secret-injected keys; in that case a strategy may still be selected and verify-env attempts verification per contract, failing at verifier init if the key is still missing.

Verification strategy is chosen only from domain network config; CLDF does not maintain chain allowlists.
Requires a domain-specific ContractInputsProvider to supply contract metadata.`

	verifyEnvExample = `
		# Verify all EVM contracts in staging
		<domain> contract verify-env -e staging

		# Verify using local datastore files only (ignore domain config)
		<domain> contract verify-env -e staging --local

		# Verify only specific networks
		<domain> contract verify-env -e staging -n 5719461335882077547,1216300075444106652

		# Verify a specific contract address
		<domain> contract verify-env -e staging -a 0x6871D60d0C721B3813bb38a37DbCe5E84cEfBB3B`
)

func newVerifyEnvCmdWithUse(cfg Config, use string) *cobra.Command {
	var (
		filterNetworks string
		filterContract string
		pollInterval   time.Duration
		rateLimitDelay time.Duration
		customDomain   string
		fromLocal      bool
	)

	cmd := &cobra.Command{
		Use:     use,
		Short:   verifyEnvShort,
		Long:    verifyEnvLong,
		Example: verifyEnvExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			env, _ := cmd.Flags().GetString("environment")
			dom := cfg.Domain
			if customDomain != "" {
				d, err := domain.GetDomain(customDomain)
				if err != nil {
					return fmt.Errorf("invalid domain %q: %w", customDomain, err)
				}
				dom = d
			}
			f := verifyEnvFlags{
				environment:    env,
				filterNetworks: filterNetworks,
				filterContract: filterContract,
				pollInterval:   pollInterval,
				rateLimitDelay: rateLimitDelay,
				fromLocal:      fromLocal,
			}

			return runVerifyEnv(cmd, cfg, f, dom)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().StringVarP(&filterNetworks, "networks", "n", "", "Optional comma-separated list of chain selectors to verify")
	cmd.Flags().StringVarP(&filterContract, "address", "a", "", "Optional contract address to verify")
	cmd.Flags().DurationVarP(&pollInterval, "poll-interval", "p", 5*time.Second, "Polling interval for verification status")
	cmd.Flags().DurationVar(&rateLimitDelay, "rate-limit-delay", 1*time.Second, "Delay between contract verifications (rate limiting)")
	cmd.Flags().StringVarP(&customDomain, "domain", "d", "", "Domain to verify (default: from config)")
	cmd.Flags().BoolVar(&fromLocal, "local", false, "Use local datastore files only; ignore domain config (use for local runs)")

	return cmd
}

type verifyEnvFlags struct {
	environment    string
	filterNetworks string
	filterContract string
	pollInterval   time.Duration
	rateLimitDelay time.Duration
	fromLocal      bool
}

func runVerifyEnv(cmd *cobra.Command, cfg Config, f verifyEnvFlags, dom domain.Domain) error {
	cfg.deps()
	deps := cfg.Deps

	cfg.Logger.Infof("Verify EVM contracts for %s in environment: %s\n", dom, f.environment)

	networkCfg, err := deps.NetworkLoader(f.environment, dom)
	if err != nil {
		return fmt.Errorf("failed to load network configuration: %w", err)
	}

	networkCfg = networkCfg.FilterWith(
		cfgnet.ChainFamilyFilter(chain_selectors.FamilyEVM),
	)

	envdir := dom.EnvDir(f.environment)
	ds, err := deps.DataStoreLoader(cmd.Context(), envdir, cfg.Logger, DataStoreLoadOptions{FromLocal: f.fromLocal})
	if err != nil {
		return fmt.Errorf("failed to get datastore: %w", err)
	}

	var (
		totalContracts  int
		failed          []string // safe summary strings only (no sensitive data)
		skippedNetworks []string
	)

	allowedNetworks := parseNetworkFilter(f.filterNetworks)

	for _, network := range networkCfg.Networks() {
		chain, ok := chain_selectors.ChainBySelector(network.ChainSelector)
		if !ok {
			cfg.Logger.Debugf("No chain found for selector %d, skipping", network.ChainSelector)
			continue
		}

		if allowedNetworks != nil {
			if _, ok := allowedNetworks[chain.Selector]; !ok {
				cfg.Logger.Debugf("Skipping %s due to network filter", chain.Name)

				continue
			}
		}

		addresses := ds.Addresses().Filter(
			datastore.AddressRefByChainSelector(chain.Selector),
		)

		strategy := evm.GetVerificationStrategyForNetwork(network)
		if strategy == evm.StrategyUnknown {
			cfg.Logger.Warnf("No verification strategy found for %s, skipping network", chain.Name)
			skippedNetworks = append(skippedNetworks, chain.Name)

			continue
		}

		for _, ref := range addresses {
			if f.filterContract != "" && !strings.EqualFold(ref.Address, f.filterContract) {
				continue
			}
			if ref.Version == nil {
				cfg.Logger.Warnf("No version found for address %s on chain %s, skipping", ref.Address, chain.Name)
				continue
			}

			metadata, err := cfg.ContractInputsProvider.GetInputs(ref.Type, ref.Version)
			if err != nil {
				cfg.Logger.Debugf("Skipping %s - domain does not support contract type %s version %s: %s", ref.Address, ref.Type, ref.Version, err)
				continue
			}

			verifier, err := evm.NewVerifier(strategy, evm.VerifierConfig{
				Chain:        chain,
				Network:      network,
				Address:      ref.Address,
				Metadata:     metadata,
				ContractType: string(ref.Type),
				Version:      ref.Version.String(),
				PollInterval: f.pollInterval,
				Logger:       cfg.Logger,
				HTTPClient:   cfg.VerifierHTTPClient,
			})
			if err != nil {
				cfg.Logger.Warnf("Failed to create verifier for %s %s (%s on %s): %s",
					ref.Type, ref.Version, ref.Address, chain.Name, err)
				totalContracts++
				failed = append(failed, fmt.Sprintf("%s %s (%s on %s) - verifier init failed", ref.Type, ref.Version, ref.Address, chain.Name))

				continue
			}

			totalContracts++
			cfg.Logger.Infof("Verifying %s %s (%s on %s)", ref.Type, ref.Version, ref.Address, chain.Name)

			if err := verifier.Verify(cmd.Context()); err != nil {
				failed = append(failed, fmt.Sprintf("%s %s (%s on %s)", ref.Type, ref.Version, ref.Address, chain.Name))
				cfg.Logger.Errorf("Failed to verify %s %s (%s on %s): %s",
					ref.Type, ref.Version, ref.Address, chain.Name, err)
			}

			if ctx := cmd.Context(); ctx != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(f.rateLimitDelay):
				}
			} else {
				time.Sleep(f.rateLimitDelay)
			}
		}
	}

	cfg.Logger.Infof("\n=== Verification Summary ===\nSuccessful: %d", totalContracts-len(failed))
	if len(failed) > 0 {
		cfg.Logger.Infof("Failed: %d", len(failed))
		for _, s := range failed {
			cfg.Logger.Infof("  - %s", s)
		}
	}
	if len(skippedNetworks) > 0 {
		cfg.Logger.Infof("Networks skipped (no verifier): %v", skippedNetworks)
	}

	return nil
}

func parseNetworkFilter(s string) map[uint64]struct{} {
	if s == "" {
		return nil
	}
	m := make(map[uint64]struct{})
	for _, n := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(n)
		selector, err := strconv.ParseUint(trimmed, 10, 64)
		if err != nil {
			continue
		}
		m[selector] = struct{}{}
	}

	return m
}
