package migrations

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
)

// Nodes represents a set of node IDs. This is used to keep track of which nodes
// are available for a given domain. This struct is serialized to JSON and
// stored in each domain directory in the `nodes.json` file
type Nodes struct {
	Nodes map[string]struct{} `json:"nodes"`
}

// NewNodes creates a new Nodes struct with the given node IDs.
func NewNodes(nodeIDs []string) *Nodes {
	nodes := &Nodes{
		Nodes: make(map[string]struct{}),
	}

	for _, id := range nodeIDs {
		nodes.Nodes[id] = struct{}{}
	}

	return nodes
}

// Add adds a node ID to the set. It is No-op if already exists.
func (n *Nodes) Add(nodeID string) {
	n.Nodes[nodeID] = struct{}{}
}

// Keys returns the node IDs as a slice.
func (n *Nodes) Keys() []string {
	return slices.Collect(maps.Keys(n.Nodes))
}

// LoadNodesFromFile loads nodes from a JSON file at the specified path.
func LoadNodesFromFile(filePath string) (*Nodes, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var nodes Nodes
	if err = json.Unmarshal(data, &nodes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON for nodes: %w", err)
	}

	return &nodes, nil
}

// SaveToFile saves the nodes to a JSON file at the specified path.
// If the file already exists, the new nodes will be merged with the existing nodes.
// If a node with the same id already exists, it will be overwritten.
func (n *Nodes) SaveToFile(filePath string) error {
	_, err := os.Stat(filePath)

	// if the file already exists, load the existing nodes and merge with the new nodes
	// if the node already exists, overwrite it
	if err == nil {
		existingNodes, err2 := LoadNodesFromFile(filePath)
		if err2 != nil {
			return err2
		}

		// Add existing nodes to current nodes (current nodes take precedence)
		for _, enodeKey := range existingNodes.Keys() {
			n.Add(enodeKey)
		}
	}

	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return os.WriteFile(filePath, data, 0600)
}

// SaveNodeIDsToFile creates a new Nodes struct from node IDs and saves it to the specified path.
func SaveNodeIDsToFile(filePath string, nodeIDs []string) error {
	nodes := NewNodes(nodeIDs)
	return nodes.SaveToFile(filePath)
}
