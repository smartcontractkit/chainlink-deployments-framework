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

const rateLimitDelay = 1 * time.Second

var (
	verifyEnvShort = "Verify EVM contracts in an environment"

	verifyEnvLong = `Verify all EVM contracts in a given environment using the datastore.

If a network does not have a verification strategy defined, we ignore it and do not attempt to verify its contracts.
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
				dom = domain.MustGetDomain(customDomain)
			}
			f := verifyEnvFlags{
				environment:    env,
				filterNetworks: filterNetworks,
				filterContract: filterContract,
				pollInterval:   pollInterval,
				fromLocal:      fromLocal,
			}

			return runVerifyEnv(cmd, cfg, f, dom)
		},
	}

	flags.Environment(cmd)
	cmd.Flags().StringVarP(&filterNetworks, "networks", "n", "", "Optional comma-separated list of chain selectors to verify")
	cmd.Flags().StringVarP(&filterContract, "address", "a", "", "Optional contract address to verify")
	cmd.Flags().DurationVarP(&pollInterval, "poll-interval", "p", 5*time.Second, "Polling interval for verification status")
	cmd.Flags().StringVarP(&customDomain, "domain", "d", "", "Domain to verify (default: from config)")
	cmd.Flags().BoolVar(&fromLocal, "local", false, "Use local datastore files only; ignore domain config (use for local runs)")

	return cmd
}

type verifyEnvFlags struct {
	environment    string
	filterNetworks string
	filterContract string
	pollInterval   time.Duration
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

		strategy := evm.GetVerificationStrategy(chain.EvmChainID)
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
				cfg.Logger.Debugf("Failed to create verifier for %s on %s: %s", ref.Address, chain.Name, err)
				continue
			}

			totalContracts++
			cfg.Logger.Infof("Verifying %s %s (%s on %s)", ref.Type, ref.Version, ref.Address, chain.Name)

			if err := verifier.Verify(cmd.Context()); err != nil {
				failed = append(failed, fmt.Sprintf("%s %s (%s on %s)", ref.Type, ref.Version, ref.Address, chain.Name))
				cfg.Logger.Errorf("Failed to verify %s %s (%s on %s): %s",
					ref.Type, ref.Version, ref.Address, chain.Name, err)
			}

			time.Sleep(rateLimitDelay)
		}
	}

	fmt.Println("\n=== Verification Summary ===")
	fmt.Printf("Successful Verifications: %d\n\n", totalContracts-len(failed))
	if len(failed) > 0 {
		fmt.Printf("Failed Verifications: %d\n", len(failed))
		for _, s := range failed {
			fmt.Println("-", s)
		}
		fmt.Println()
	}
	if len(skippedNetworks) > 0 {
		fmt.Println("Networks skipped due to missing verification strategy:")
		for _, n := range skippedNetworks {
			fmt.Println("-", n)
		}
		fmt.Println()
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
