package internal

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/types"
)

// dependencyGraph represents a directed acyclic graph of analyzer dependencies
type dependencyGraph struct {
	nodes map[string]*graphNode
}

type graphNode struct {
	analyzer     types.BaseAnalyzer
	dependencies []*graphNode
	dependents   []*graphNode
}

// NewDependencyGraph creates a new dependency graph from a list of analyzers
func NewDependencyGraph(analyzers []types.BaseAnalyzer) (*dependencyGraph, error) {
	graph := &dependencyGraph{
		nodes: make(map[string]*graphNode),
	}

	// First pass: create nodes for all analyzers
	for _, a := range analyzers {
		if a == nil {
			continue
		}
		id := a.ID()
		if id == "" {
			return nil, fmt.Errorf("analyzer must have a non-empty ID")
		}
		if _, exists := graph.nodes[id]; exists {
			return nil, fmt.Errorf("duplicate analyzer ID: %s", id)
		}
		graph.nodes[id] = &graphNode{
			analyzer:     a,
			dependencies: []*graphNode{},
			dependents:   []*graphNode{},
		}
	}

	// Second pass: build dependency edges
	for _, node := range graph.nodes {
		depIDs := node.analyzer.Dependencies()
		for _, depID := range depIDs {
			if depID == "" {
				continue
			}
			depNode, exists := graph.nodes[depID]
			if !exists {
				return nil, fmt.Errorf("analyzer %s depends on unknown analyzer %s", node.analyzer.ID(), depID)
			}
			node.dependencies = append(node.dependencies, depNode)
			depNode.dependents = append(depNode.dependents, node)
		}
	}

	// Detect cycles
	if err := graph.detectCycles(); err != nil {
		return nil, err
	}

	return graph, nil
}

// detectCycles checks for circular dependencies using DFS
func (g *dependencyGraph) detectCycles() error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for id, node := range g.nodes {
		if !visited[id] {
			if err := g.detectCyclesDFS(node, visited, recStack, []string{}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *dependencyGraph) detectCyclesDFS(node *graphNode, visited, recStack map[string]bool, path []string) error {
	id := node.analyzer.ID()
	visited[id] = true
	recStack[id] = true
	path = append(path, id)

	for _, dep := range node.dependencies {
		depID := dep.analyzer.ID()
		if !visited[depID] {
			if err := g.detectCyclesDFS(dep, visited, recStack, path); err != nil {
				return err
			}
		} else if recStack[depID] {
			// Found a cycle
			cyclePath := append(path, depID)
			return fmt.Errorf("circular dependency detected: %v", cyclePath)
		}
	}

	recStack[id] = false
	return nil
}

// TopologicalSort returns analyzers in execution order (dependencies first)
func (g *dependencyGraph) TopologicalSort() ([]types.BaseAnalyzer, error) {
	result := []types.BaseAnalyzer{}
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(*graphNode) error
	visit = func(node *graphNode) error {
		id := node.analyzer.ID()
		if temp[id] {
			return fmt.Errorf("cycle detected at %s", id)
		}
		if visited[id] {
			return nil
		}

		temp[id] = true
		for _, dep := range node.dependencies {
			if err := visit(dep); err != nil {
				return err
			}
		}
		temp[id] = false
		visited[id] = true
		result = append(result, node.analyzer)
		return nil
	}

	for _, node := range g.nodes {
		if !visited[node.analyzer.ID()] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// getLevels returns analyzers grouped by execution level (for parallel execution)
func (g *dependencyGraph) getLevels() [][]types.BaseAnalyzer {
	inDegree := make(map[string]int)
	for id, node := range g.nodes {
		inDegree[id] = len(node.dependencies)
	}

	var levels [][]types.BaseAnalyzer
	remaining := len(g.nodes)

	for remaining > 0 {
		var currentLevel []types.BaseAnalyzer
		for id, node := range g.nodes {
			if inDegree[id] == 0 {
				currentLevel = append(currentLevel, node.analyzer)
			}
		}

		if len(currentLevel) == 0 {
			// Should not happen if cycle detection worked
			break
		}

		levels = append(levels, currentLevel)

		// Remove nodes in current level and update in-degrees
		for _, a := range currentLevel {
			id := a.ID()
			inDegree[id] = -1 // Mark as processed
			remaining--

			node := g.nodes[id]
			for _, dependent := range node.dependents {
				depID := dependent.analyzer.ID()
				if inDegree[depID] > 0 {
					inDegree[depID]--
				}
			}
		}
	}

	return levels
}
