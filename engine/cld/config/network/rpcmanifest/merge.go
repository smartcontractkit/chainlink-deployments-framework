package rpcmanifest

import (
	"errors"
	"fmt"
	"slices"

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
//   - If YAML rpcs is non-empty and rpcs_mode is "replace" (the default), YAML rpcs are kept.
//   - If YAML rpcs is non-empty and rpcs_mode is "append", YAML rpcs come first, followed by the
//     manifest's provider RPCs (skipping any with the same HTTP URL).
//   - Otherwise rpcs are populated from the manifest (one provider-specific proxy URL per node;
//     the load-balanced chain-level URL is not used).
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
	appendMode := len(network.RPCs) > 0 && network.RPCsMode == cfgnet.RPCsModeAppend
	if len(network.RPCs) == 0 || appendMode {
		if !inManifest {
			if opts.SkipMissingManifestChains {
				lggr.Warnw("chain not present in rpc manifest",
					"chain_selector", network.ChainSelector,
				)

				// A chain with yaml rpcs is still usable without the manifest.
				return network, appendMode, nil
			}

			return cfgnet.Network{}, false, fmt.Errorf(
				"chain selector %d uses the rpc manifest (empty rpcs or rpcs_mode: append) but is not present in rpc manifest",
				network.ChainSelector,
			)
		}

		discovered := RPCsForChain(network.ChainSelector, entry)
		if len(discovered) == 0 && !appendMode {
			return cfgnet.Network{}, false, fmt.Errorf(
				"chain selector %d has no rpcs in YAML and no provider nodes in rpc manifest",
				network.ChainSelector,
			)
		}

		network.RPCs = mergeRPCs(network.RPCs, discovered)
		lggr.Infow("populated rpcs from manifest",
			"chain_selector", network.ChainSelector,
			"rpc_count", len(network.RPCs),
		)
	}
	// The result is a final list; clear the mode so re-writing it as yaml doesn't append twice.
	network.RPCsMode = ""

	return network, true, nil
}

// mergeRPCs returns the yaml RPCs followed by any discovered RPCs whose HTTP URL isn't already
// listed.
func mergeRPCs(yaml, discovered []cfgnet.RPC) []cfgnet.RPC {
	out := slices.Clone(yaml)
	for _, rpc := range discovered {
		if !slices.ContainsFunc(out, func(r cfgnet.RPC) bool { return r.HTTPURL == rpc.HTTPURL }) {
			out = append(out, rpc)
		}
	}

	return out
}
