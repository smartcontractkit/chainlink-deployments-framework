package scheduler

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer"
)

// RunFunc is invoked for each analyzer when Graph.Run executes.
type RunFunc[T analyzer.BaseAnalyzer] func(ctx context.Context, analyzer T) error

// Graph stores analyzers and their execution layers in dependency order.
type Graph[T analyzer.BaseAnalyzer] struct {
	byID   map[string]T
	levels [][]string
}

// New validates analyzers and creates a dependency graph.
func New[T analyzer.BaseAnalyzer](analyzers []T) (*Graph[T], error) {
	byID := make(map[string]T, len(analyzers))
	inDegree := make(map[string]int, len(analyzers))
	dependents := make(map[string][]string, len(analyzers))

	for _, a := range analyzers {
		id := a.ID()
		if id == "" {
			return nil, errors.New("analyzer ID cannot be empty")
		}
		if _, exists := byID[id]; exists {
			return nil, fmt.Errorf("duplicate analyzer ID %q", id)
		}
		byID[id] = a
	}

	for _, a := range analyzers {
		id := a.ID()
		seenDeps := make(map[string]struct{}, len(a.Dependencies()))
		for _, depID := range a.Dependencies() {
			if depID == "" {
				return nil, fmt.Errorf("analyzer %q has empty dependency ID", id)
			}
			if depID == id {
				return nil, fmt.Errorf("analyzer %q cannot depend on itself", id)
			}
			if _, exists := byID[depID]; !exists {
				return nil, fmt.Errorf("analyzer %q depends on unknown analyzer %q", id, depID)
			}
			if _, duplicate := seenDeps[depID]; duplicate {
				continue
			}
			seenDeps[depID] = struct{}{}

			inDegree[id]++
			dependents[depID] = append(dependents[depID], id)
		}
	}

	queue := make([]string, 0, len(analyzers))
	for id := range byID {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	slices.Sort(queue)

	levels := make([][]string, 0)
	processed := 0

	for len(queue) > 0 {
		currentLevel := make([]string, len(queue))
		copy(currentLevel, queue)
		levels = append(levels, currentLevel)
		processed += len(currentLevel)

		next := make([]string, 0)
		for _, id := range currentLevel {
			for _, dependentID := range dependents[id] {
				inDegree[dependentID]--
				if inDegree[dependentID] == 0 {
					next = append(next, dependentID)
				}
			}
		}
		slices.Sort(next)
		queue = next
	}

	if processed != len(analyzers) {
		return nil, errors.New("analyzer dependency graph contains a cycle")
	}

	return &Graph[T]{
		byID:   byID,
		levels: levels,
	}, nil
}

// Levels returns analyzer IDs grouped by dependency level.
func (g *Graph[T]) Levels() [][]string {
	result := make([][]string, len(g.levels))
	for i := range g.levels {
		level := make([]string, len(g.levels[i]))
		copy(level, g.levels[i])
		result[i] = level
	}

	return result
}

// Run executes analyzers in dependency order and in parallel per level.
//
// Execution is best-effort:
//   - analyzer failures do not stop other independent analyzers
//   - analyzers whose dependencies failed are skipped
//   - all failures are returned as a single aggregated error
func (g *Graph[T]) Run(ctx context.Context, run RunFunc[T]) error {
	if run == nil {
		return errors.New("run function cannot be nil")
	}

	var (
		mu        sync.Mutex
		runErrors []error
		failedIDs = make(map[string]struct{}, len(g.byID))
	)
	appendRunError := func(err error) {
		mu.Lock()
		runErrors = append(runErrors, err)
		mu.Unlock()
	}

	for _, level := range g.levels {
		if err := ctx.Err(); err != nil {
			appendRunError(err)
			break
		}

		var (
			wg            sync.WaitGroup
			levelFailedMu sync.Mutex
			levelFailed   []string
		)

		for _, id := range level {
			analyzer := g.byID[id]
			hasFailedDep := false
			for _, depID := range analyzer.Dependencies() {
				if _, failed := failedIDs[depID]; failed {
					hasFailedDep = true
					break
				}
			}

			if hasFailedDep {
				skipErr := fmt.Errorf("skip analyzer %q: dependency failure", id)
				appendRunError(skipErr)
				failedIDs[id] = struct{}{}

				continue
			}

			wg.Add(1)
			analyzerID := id
			go func() {
				defer wg.Done()
				if err := run(ctx, g.byID[analyzerID]); err != nil {
					appendRunError(fmt.Errorf("run analyzer %q: %w", analyzerID, err))

					levelFailedMu.Lock()
					levelFailed = append(levelFailed, analyzerID)
					levelFailedMu.Unlock()
				}
			}()
		}
		wg.Wait()

		for _, failedID := range levelFailed {
			failedIDs[failedID] = struct{}{}
		}
	}

	if len(runErrors) > 0 {
		return errors.Join(runErrors...)
	}

	return nil
}
