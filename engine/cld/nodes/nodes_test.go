package migrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Nodes_Add(t *testing.T) {
	t.Parallel()

	// Check initial
	got := NewNodes([]string{"a", "b"})
	require.Len(t, got.Nodes, 2)
	require.Contains(t, got.Nodes, "a")
	require.Contains(t, got.Nodes, "b")

	// Add a new node
	got.Add("c")
	require.Len(t, got.Nodes, 3)
	require.Contains(t, got.Nodes, "c")

	// Add an existing node should not change anything
	got.Add("c")
	require.Len(t, got.Nodes, 3)
	require.Contains(t, got.Nodes, "c")
}

func Test_Nodes_Keys(t *testing.T) {
	t.Parallel()

	got := NewNodes([]string{"a", "b", "c"})
	require.ElementsMatch(t, got.Keys(), []string{"a", "b", "c"})
}

func Test_LoadNodesFromFile(t *testing.T) {
	t.Parallel()

	t.Run("loads nodes from valid JSON file", func(t *testing.T) {
		t.Parallel()

		// Create a temporary file with valid nodes JSON
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		expectedNodes := &Nodes{
			Nodes: map[string]struct{}{
				"node1": {},
				"node2": {},
				"node3": {},
			},
		}

		data, err := json.MarshalIndent(expectedNodes, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filePath, data, 0600))

		// Load nodes from file
		nodes, err := LoadNodesFromFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, nodes)
		require.ElementsMatch(t, nodes.Keys(), []string{"node1", "node2", "node3"})
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		t.Parallel()

		nodes, err := LoadNodesFromFile("non-existent-file.json")
		require.Error(t, err)
		require.Nil(t, nodes)
		require.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "invalid.json")

		// Write invalid JSON
		require.NoError(t, os.WriteFile(filePath, []byte("invalid json"), 0600))

		nodes, err := LoadNodesFromFile(filePath)
		require.Error(t, err)
		require.Nil(t, nodes)
		require.Contains(t, err.Error(), "failed to unmarshal JSON for nodes")
	})
}

func Test_Nodes_SaveToFile(t *testing.T) {
	t.Parallel()

	t.Run("saves nodes to new file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		nodes := NewNodes([]string{"node1", "node2", "node3"})

		err := nodes.SaveToFile(filePath)
		require.NoError(t, err)

		// Verify file was created and contains correct data
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var savedNodes Nodes
		require.NoError(t, json.Unmarshal(data, &savedNodes))
		require.ElementsMatch(t, savedNodes.Keys(), []string{"node1", "node2", "node3"})
	})

	t.Run("merges with existing file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		// Create existing nodes file
		existingNodes := NewNodes([]string{"existing1", "existing2"})
		err := existingNodes.SaveToFile(filePath)
		require.NoError(t, err)

		// Save new nodes (should merge with existing)
		newNodes := NewNodes([]string{"new1", "new2", "existing1"}) // existing1 should not be duplicated
		err = newNodes.SaveToFile(filePath)
		require.NoError(t, err)

		// Verify merged result
		loadedNodes, err := LoadNodesFromFile(filePath)
		require.NoError(t, err)
		require.ElementsMatch(t, loadedNodes.Keys(), []string{"existing1", "existing2", "new1", "new2"})
	})

	t.Run("handles file permission errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Try to save to a directory that doesn't exist
		nodes := NewNodes([]string{"node1"})
		err := nodes.SaveToFile("/non-existent-dir/nodes.json")
		require.Error(t, err)
	})
}

func Test_SaveNodeIDsToFile(t *testing.T) {
	t.Parallel()

	t.Run("creates nodes and saves to file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		nodeIDs := []string{"node1", "node2", "node3"}
		err := SaveNodeIDsToFile(filePath, nodeIDs)
		require.NoError(t, err)

		// Verify file was created and contains correct data
		loadedNodes, err := LoadNodesFromFile(filePath)
		require.NoError(t, err)
		require.ElementsMatch(t, loadedNodes.Keys(), nodeIDs)
	})

	t.Run("merges with existing file", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		// Create existing file
		err := SaveNodeIDsToFile(filePath, []string{"existing1", "existing2"})
		require.NoError(t, err)

		// Save new nodes
		err = SaveNodeIDsToFile(filePath, []string{"new1", "new2", "existing1"})
		require.NoError(t, err)

		// Verify merged result
		loadedNodes, err := LoadNodesFromFile(filePath)
		require.NoError(t, err)
		require.ElementsMatch(t, loadedNodes.Keys(), []string{"existing1", "existing2", "new1", "new2"})
	})

	t.Run("handles empty node IDs", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "nodes.json")

		err := SaveNodeIDsToFile(filePath, []string{})
		require.NoError(t, err)

		loadedNodes, err := LoadNodesFromFile(filePath)
		require.NoError(t, err)
		require.Empty(t, loadedNodes.Keys())
	})
}
