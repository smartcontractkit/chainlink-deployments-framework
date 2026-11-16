package rpcmanifest

import (
	"fmt"
	"strings"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const preferredTransportScheme = "http"

// RPCsForChain builds one network RPC entry per provider node in a manifest chain entry. It
// deliberately does not include the chain-level load-balanced proxy URL
// (https://rpcs.cldev.sh/{selector}): CLD wants to talk to a specific provider, not whichever
// one the load balancer picks. Nodes without a numerical ID have no proxy path and are skipped.
func RPCsForChain(selector uint64, entry ChainEntry) []cfgnet.RPC {
	rpcs := make([]cfgnet.RPC, 0, len(entry.Nodes))

	for _, node := range entry.Nodes {
		numericalID := node.ID.Numerical
		if numericalID == "" {
			continue
		}

		rpcs = append(rpcs, cfgnet.RPC{
			RPCName:            fmt.Sprintf("%s (%s)", node.Provider, numericalID),
			PreferredURLScheme: preferredTransportScheme,
			HTTPURL:            proxyHTTPURL(selector, numericalID),
			WSURL:              proxyWSURL(selector, numericalID),
		})
	}

	return rpcs
}

func proxyHTTPURL(selector uint64, numericalID string) string {
	if numericalID == "" {
		return fmt.Sprintf("%s/%d", rpcProxyBase, selector)
	}

	return fmt.Sprintf("%s/%d/%s", rpcProxyBase, selector, numericalID)
}

func proxyWSURL(selector uint64, numericalID string) string {
	httpURL := proxyHTTPURL(selector, numericalID)
	if after, ok := strings.CutPrefix(httpURL, "https://"); ok {
		return "wss://" + after
	}
	if after, ok := strings.CutPrefix(httpURL, "http://"); ok {
		return "ws://" + after
	}

	return httpURL
}
