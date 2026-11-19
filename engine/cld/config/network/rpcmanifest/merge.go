package rpcmanifest

import (
	"errors"
	"fmt"
	"slices"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// MergeInto returns a copy of yamlCfg with each network's rpcs resolved against the manifest:
//   - non-empty rpcs in "replace" mode (the default) are kept as-is;
//   - non-empty rpcs in "append" mode are followed by the manifest's provider RPCs, skipping any
//     with an HTTP URL already listed;
//   - empty rpcs are filled from the manifest's provider RPCs.
//
// The manifest only ever supplies rpcs: it never adds networks or changes other fields.
func MergeInto(yamlCfg *cfgnet.Config, manifest *Manifest, lggr logger.Logger) (*cfgnet.Config, error) {
	if yamlCfg == nil {
		return nil, errors.New("yaml network config is nil")
	}
	if manifest == nil {
		return nil, errors.New("rpc manifest is nil")
	}

	networks := yamlCfg.Networks()
	for i, network := range networks {
		merged, err := mergeNetwork(network, manifest, lggr)
		if err != nil {
			return nil, err
		}
		networks[i] = merged
	}

	out := cfgnet.NewConfig(networks)
	if err := out.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate merged network config: %w", err)
	}

	return out, nil
}

func mergeNetwork(network cfgnet.Network, manifest *Manifest, lggr logger.Logger) (cfgnet.Network, error) {
	if err := network.ValidateStructure(); err != nil {
		return cfgnet.Network{}, fmt.Errorf("failed to validate network %d: %w", network.ChainSelector, err)
	}

	appendMode := network.RPCsMode == cfgnet.RPCsModeAppend
	// The merged list is final; clearing the mode keeps a second merge from appending again.
	network.RPCsMode = ""

	if len(network.RPCs) > 0 && !appendMode {
		return network, nil
	}

	entry, ok := manifest.Chain(network.ChainSelector)
	if !ok {
		return cfgnet.Network{}, fmt.Errorf(
			"chain selector %d uses the rpc manifest (empty rpcs or rpcs_mode: append) but is not present in rpc manifest",
			network.ChainSelector,
		)
	}

	discovered := RPCsForChain(network.ChainSelector, entry)
	if len(network.RPCs) == 0 && len(discovered) == 0 {
		return cfgnet.Network{}, fmt.Errorf(
			"chain selector %d has no rpcs in YAML and no provider nodes in rpc manifest",
			network.ChainSelector,
		)
	}

	network.RPCs = appendUniqueRPCs(network.RPCs, discovered)
	lggr.Infow("populated rpcs from manifest",
		"chain_selector", network.ChainSelector,
		"rpc_count", len(network.RPCs),
	)

	return network, nil
}

// appendUniqueRPCs returns rpcs followed by each of extra whose HTTP URL is not already listed.
func appendUniqueRPCs(rpcs, extra []cfgnet.RPC) []cfgnet.RPC {
	out := slices.Clone(rpcs)
	for _, rpc := range extra {
		if !slices.ContainsFunc(out, func(r cfgnet.RPC) bool { return r.HTTPURL == rpc.HTTPURL }) {
			out = append(out, rpc)
		}
	}

	return out
}
