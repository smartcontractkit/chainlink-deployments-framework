package offchain

// NodeKey is the key type to use to find a node
type NodeKey string

const (
	NodeKey_ID     NodeKey = "id"
	NodeKey_CSAKey NodeKey = "csa_key"
	NodeKey_Name   NodeKey = "name"
	NodeKey_Label  NodeKey = "label"
)
