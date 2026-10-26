package rpcmanifest

import (
	"fmt"
	"strings"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

const (
	cllProxyRPCName          = "CLL Proxy"
	preferredTransportScheme = "http"
)

// RPCsForChain builds network RPC entries from a manifest chain entry.
func RPCsForChain(selector uint64, entry ChainEntry) []cfgnet.RPC {
	rpcs := make([]cfgnet.RPC, 0, 1+len(entry.Nodes))

	rpcs = append(rpcs, cfgnet.RPC{
		RPCName:            cllProxyRPCName,
		PreferredURLScheme: preferredTransportScheme,
		HTTPURL:            proxyHTTPURL(selector, ""),
		WSURL:              proxyWSURL(selector, ""),
	})

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
