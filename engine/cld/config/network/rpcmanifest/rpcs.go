package rpcmanifest

import (
	"fmt"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const (
	rpcProxyHost             = "rpcs.cldev.sh"
	preferredTransportScheme = "http"
)

// RPCsForChain returns one RPC per provider node of a manifest chain entry, each addressing that
// provider directly rather than the load-balanced chain URL. Nodes without a numerical ID have no
// proxy path and are skipped.
func RPCsForChain(selector uint64, entry ChainEntry) []cfgnet.RPC {
	rpcs := make([]cfgnet.RPC, 0, len(entry.Nodes))

	for _, node := range entry.Nodes {
		id := node.ID.Numerical
		if id == "" {
			continue
		}

		rpcs = append(rpcs, cfgnet.RPC{
			RPCName:            fmt.Sprintf("%s (%s)", node.Provider, id),
			PreferredURLScheme: preferredTransportScheme,
			HTTPURL:            fmt.Sprintf("https://%s/%d/%s", rpcProxyHost, selector, id),
			WSURL:              fmt.Sprintf("wss://%s/%d/%s", rpcProxyHost, selector, id),
		})
	}

	return rpcs
}
