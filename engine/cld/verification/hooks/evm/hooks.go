// Package evm provides changeset lifecycle hooks for EVM contract verification.
package evm

import (
	"context"
	"errors"
	"fmt"
	"time"

	chain_selectors "github.com/smartcontractkit/chain-selectors"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldverification "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	verifyDeployedContractsHookName     = "verify-deployed-contracts"
	requireVerifiedEnvContractsHookName = "require-verified-env-contracts"
	defaultVerifyPollInterval           = 5 * time.Second
	defaultVerifyRateLimitDelay         = 1 * time.Second
)

// interVerifyDelay is the pause between explorer calls in iterateEVMVerifiers. Tests set this to 0 in init.
var interVerifyDelay = defaultVerifyRateLimitDelay

// NewVerifyDeployedEVMContractsPostHook returns a post-hook that verifies EVM contracts
// newly written to ChangesetOutput.DataStore, using the same ContractInputsProvider path as
// contract verify-env flows.
//
// It runs only after a successful Apply. Hook failures use Abort policy and fail the pipeline.
func NewVerifyDeployedEVMContractsPostHook(dom domain.Domain, provider evm.ContractInputsProvider) changeset.PostHook {
	return changeset.PostHook{
		HookDefinition: changeset.HookDefinition{
			Name:          verifyDeployedContractsHookName,
			FailurePolicy: changeset.Abort,
			Timeout:       30 * time.Minute,
		},
		Func: verifyDeployedContracts(dom, provider),
	}
}

func verifyDeployedContracts(dom domain.Domain, provider evm.ContractInputsProvider) changeset.PostHookFunc {
	return func(ctx context.Context, params changeset.PostHookParams) error {
		if params.Err != nil {
			return nil
		}
		if params.Output.DataStore == nil {
			return nil
		}

		networkCfg, err := loadFilteredEVMNetworks(params.Env.Name, dom, params.Env.Logger)
		if err != nil {
			return fmt.Errorf("verify hook: load networks: %w", err)
		}

		return iterateEVMVerifiers(ctx, params.Output.DataStore.Seal(), networkCfg, provider, params.Env.Logger, "verify hook",
			func(ctx context.Context, v cldverification.Verifiable, ref datastore.AddressRef, ch chain_selectors.Chain) error {
				params.Env.Logger.Infof("verify hook: verifying %s %s (%s on %s)", ref.Type, ref.Version, ref.Address, ch.Name)
				if err := v.Verify(ctx); err != nil {
					return fmt.Errorf("verify hook: verifier %s for %s %s (%s on %s): %w", v.String(), ref.Type, ref.Version, ref.Address, ch.Name, err)
				}
				return nil
			},
		)
	}
}

// NewRequireVerifiedEVMContractsPreHook returns a global pre-hook that checks block-explorer
// verification status for EVM addresses in the environment datastore that can be resolved to a
// supported network/contract type (same ContractInputsProvider as contract verify-env). It uses
// IsVerified only (no submission). If any checked contract is not verified, the hook returns an
// error and blocks the changeset (Abort policy).
func NewRequireVerifiedEVMContractsPreHook(dom domain.Domain, provider evm.ContractInputsProvider) changeset.PreHook {
	return changeset.PreHook{
		HookDefinition: changeset.HookDefinition{
			Name:          requireVerifiedEnvContractsHookName,
			FailurePolicy: changeset.Abort,
			Timeout:       30 * time.Minute,
		},
		Func: requireVerifiedEnvContracts(dom, provider),
	}
}

func requireVerifiedEnvContracts(dom domain.Domain, provider evm.ContractInputsProvider) changeset.PreHookFunc {
	return func(ctx context.Context, params changeset.PreHookParams) error {
		ds, err := dom.EnvDir(params.Env.Name).DataStore()
		if err != nil {
			return fmt.Errorf("require verified pre-hook: load datastore: %w", err)
		}

		networkCfg, err := loadFilteredEVMNetworks(params.Env.Name, dom, params.Env.Logger)
		if err != nil {
			return fmt.Errorf("require verified pre-hook: load networks: %w", err)
		}

		return iterateEVMVerifiers(ctx, ds, networkCfg, provider, params.Env.Logger, "require verified pre-hook",
			func(ctx context.Context, v cldverification.Verifiable, ref datastore.AddressRef, ch chain_selectors.Chain) error {
				params.Env.Logger.Infof("require verified pre-hook: checking %s", v.String())
				verified, err := v.IsVerified(ctx)
				if err != nil {
					return fmt.Errorf("%s: check verified: %w", v.String(), err)
				}
				if !verified {
					return fmt.Errorf("%s: contract is not verified on explorer", v.String())
				}
				return nil
			},
		)
	}
}

func loadFilteredEVMNetworks(envName string, dom domain.Domain, lggr logger.Logger) (*cfgnet.Config, error) {
	networkCfg, err := config.LoadNetworks(envName, dom, lggr)
	if err != nil {
		return nil, err
	}
	return networkCfg.FilterWith(cfgnet.ChainFamilyFilter(chain_selectors.FamilyEVM)), nil
}

// iterateEVMVerifiers walks datastore EVM address refs, builds an explorer verifier per ref, and runs step.
func iterateEVMVerifiers(
	ctx context.Context,
	ds datastore.DataStore,
	networkCfg *cfgnet.Config,
	provider evm.ContractInputsProvider,
	lggr logger.Logger,
	logPrefix string,
	step func(ctx context.Context, v cldverification.Verifiable, ref datastore.AddressRef, ch chain_selectors.Chain) error,
) error {
	var errs []error

	for _, network := range networkCfg.Networks() {
		ch, ok := chain_selectors.ChainBySelector(network.ChainSelector)
		if !ok {
			continue
		}

		strategy := evm.GetVerificationStrategy(ch.EvmChainID)
		if strategy == evm.StrategyUnknown {
			lggr.Warnf("%s: no verification strategy for %s, skipping network", logPrefix, ch.Name)
			continue
		}

		addresses := ds.Addresses().Filter(datastore.AddressRefByChainSelector(ch.Selector))
		for _, ref := range addresses {
			if ref.Version == nil {
				lggr.Warnf("%s: no version for %s on %s, skipping", logPrefix, ref.Address, ch.Name)
				continue
			}

			metadata, err := provider.GetInputs(ref.Type, ref.Version)
			if err != nil {
				lggr.Debugf("%s: skipping %s (%s): %v", logPrefix, ref.Address, ref.Type, err)
				continue
			}

			v, err := evm.NewVerifier(strategy, evm.VerifierConfig{
				Chain:        ch,
				Network:      network,
				Address:      ref.Address,
				Metadata:     metadata,
				ContractType: string(ref.Type),
				Version:      ref.Version.String(),
				PollInterval: defaultVerifyPollInterval,
				Logger:       lggr,
			})
			if err != nil {
				errs = append(errs, fmt.Errorf("%s %s (%s on %s): %w",
					ref.Type, ref.Version, ref.Address, ch.Name, err))
				continue
			}

			if err := step(ctx, v, ref, ch); err != nil {
				errs = append(errs, err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(interVerifyDelay):
			}
		}
	}

	return errors.Join(errs...)
}
