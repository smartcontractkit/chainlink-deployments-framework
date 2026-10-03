package rpcmanifest

import (
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// MergeOptions configures optional merge behavior.
type MergeOptions struct {
	// SkipMissingManifestChains drops YAML chains that have no rpcs and are absent from the manifest
	// instead of failing. Intended for preview/generation tooling.
	SkipMissingManifestChains bool
}

// MergeInto applies RPC proxy manifest data onto a YAML-loaded network config.
//
// For each network in yamlCfg:
//   - If YAML rpcs is non-empty, YAML rpcs are kept (override).
//   - Otherwise rpcs are populated from the manifest (load-balanced proxy + per-node URLs).
//
// All other fields (block_explorer, metadata, etc.) always come from YAML, untouched — the
// manifest is only ever used to fill in rpcs.
func MergeInto(yamlCfg *cfgnet.Config, manifest *Manifest, lggr logger.Logger) (*cfgnet.Config, error) {
	return MergeIntoWithOptions(yamlCfg, manifest, lggr, MergeOptions{})
}

func MergeIntoWithOptions(
	yamlCfg *cfgnet.Config,
	manifest *Manifest,
	lggr logger.Logger,
	opts MergeOptions,
) (*cfgnet.Config, error) {
	if yamlCfg == nil {
		return nil, errors.New("yaml network config is nil")
	}
	if manifest == nil {
		return nil, errors.New("rpc manifest is nil")
	}

	merged := make([]cfgnet.Network, 0, len(yamlCfg.Networks()))
	for _, network := range yamlCfg.Networks() {
		updated, include, err := mergeNetworkWithOptions(network, manifest, lggr, opts)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		merged = append(merged, updated)
	}

	out := cfgnet.NewConfig(merged)
	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("validate merged network config: %w", err)
	}

	return out, nil
}

func mergeNetworkWithOptions(
	network cfgnet.Network,
	manifest *Manifest,
	lggr logger.Logger,
	opts MergeOptions,
) (cfgnet.Network, bool, error) {
	if network.ChainSelector == 0 {
		return cfgnet.Network{}, false, errors.New("network has no chain_selector defined in YAML")
	}

	entry, inManifest := manifest.Chain(network.ChainSelector)
	if len(network.RPCs) == 0 {
		if !inManifest {
			if opts.SkipMissingManifestChains {
				lggr.Warnw("skipping chain not present in rpc manifest",
					"chain_selector", network.ChainSelector,
				)

				return cfgnet.Network{}, false, nil
			}

			return cfgnet.Network{}, false, fmt.Errorf(
				"chain selector %d has no rpcs in YAML and is not present in rpc manifest",
				network.ChainSelector,
			)
		}
		network.RPCs = RPCsForChain(network.ChainSelector, entry)
		lggr.Infow("populated rpcs from manifest",
			"chain_selector", network.ChainSelector,
			"rpc_count", len(network.RPCs),
		)
	}

	return network, true, nil
}
