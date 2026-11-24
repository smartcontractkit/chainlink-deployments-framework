package rpcmanifest

import (
	"fmt"
	"regexp"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const preferredTransportScheme = "http"

// nodeIDPattern restricts node IDs to a single safe URL path segment.
var nodeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// rpcsForChain returns one RPC per provider node of a manifest chain entry, each addressing that
// provider directly (baseURL/selector/id) rather than the load-balanced chain URL. The manifest has
// no websocket information, so only HTTP URLs are set.
func rpcsForChain(baseURL string, selector uint64, entry chainEntry, lggr logger.Logger) []cfgnet.RPC {
	rpcs := make([]cfgnet.RPC, 0, len(entry.Nodes))

	for _, n := range entry.Nodes {
		id := n.ID.Numerical
		if !nodeIDPattern.MatchString(id) {
			lggr.Warnw("skipping rpc manifest node with invalid numerical id",
				"chain_selector", selector,
				"node_id", id,
			)

			continue
		}

		rpcs = append(rpcs, cfgnet.RPC{
			RPCName:            fmt.Sprintf("%s (%s)", n.Provider, id),
			PreferredURLScheme: preferredTransportScheme,
			HTTPURL:            fmt.Sprintf("%s/%d/%s", baseURL, selector, id),
		})
	}

	return rpcs
}
