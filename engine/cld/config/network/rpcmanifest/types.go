package rpcmanifest

const (
	// EnvRPCManifestURL overrides the domain's rpc_manifest_url (e.g. to use a local file). RPC
	// discovery is opt-in per domain; no default source is built in.
	EnvRPCManifestURL = "CLD_RPC_MANIFEST_URL"
	rpcProxyBase      = "https://rpcs.cldev.sh"
)

// ChainEntry is one chain from the RPC proxy manifest (rpcs.json).
type ChainEntry struct {
	ChainID       string `json:"ChainID"`
	ChainSelector string `json:"ChainSelector"`
	Names         Names  `json:"Names"`
	Nodes         []Node `json:"Nodes"`
}

// Node identifies a single RPC provider endpoint in the proxy.
type Node struct {
	ID       NodeID `json:"ID"`
	Provider string `json:"Provider"`
}

// NodeID holds stable and numerical identifiers for a proxy node path segment.
type NodeID struct {
	Numerical string `json:"Numerical"`
	Stable    string `json:"Stable"`
}

// Names holds human-readable chain naming metadata from the manifest.
type Names struct {
	Blockchain string `json:"Blockchain"`
	Canonical  string `json:"Canonical"`
	Fullname   string `json:"Fullname"`
	Network    string `json:"Network"`
	Shortname  string `json:"Shortname"`
}
