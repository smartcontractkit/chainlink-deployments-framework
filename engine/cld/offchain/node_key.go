package offchain

// NodeKey is the key type to use to find a node
type NodeKey string

const (
	NodeKey_ID     NodeKey = "id"      //nolint:revive // renaming would be a breaking change
	NodeKey_CSAKey NodeKey = "csa_key" //nolint:revive // renaming would be a breaking change
	NodeKey_Name   NodeKey = "name"    //nolint:revive // renaming would be a breaking change
	NodeKey_Label  NodeKey = "label"   //nolint:revive // renaming would be a breaking change
)
