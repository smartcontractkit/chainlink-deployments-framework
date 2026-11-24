package rpcmanifest

// chainEntry is one chain from the RPC proxy manifest (rpcs.json).
type chainEntry struct {
	ChainID       string `json:"ChainID"`
	ChainSelector string `json:"ChainSelector"`
	Names         names  `json:"Names"`
	Nodes         []node `json:"Nodes"`
}

// node identifies a single RPC provider endpoint in the proxy.
type node struct {
	ID       nodeID `json:"ID"`
	Provider string `json:"Provider"`
}

// nodeID holds stable and numerical identifiers for a proxy node path segment.
type nodeID struct {
	Numerical string `json:"Numerical"`
	Stable    string `json:"Stable"`
}

// names holds human-readable chain naming metadata from the manifest.
type names struct {
	Blockchain string `json:"Blockchain"`
	Canonical  string `json:"Canonical"`
	Fullname   string `json:"Fullname"`
	Network    string `json:"Network"`
	Shortname  string `json:"Shortname"`
}
