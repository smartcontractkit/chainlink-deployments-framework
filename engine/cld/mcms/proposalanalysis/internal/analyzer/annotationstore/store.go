package annotationstore

import "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/internal/analyzer/annotation"

// DependencyAnnotationStore exposes dependency-scoped annotations for an analyzer.
//
// Implementations MUST enforce that reads are limited to the current analyzer's
// declared Dependencies().
type DependencyAnnotationStore interface {
	// DependencyAnnotations returns all annotations available to the current entity.
	DependencyAnnotations() annotation.Annotations

	// Filter returns dependency annotations matching every provided predicate.
	// Predicates can be composed using:
	//   - ByLevel
	//   - ByName
	//   - ByType
	//   - ByAnalyzer
	Filter(preds ...DependencyAnnotationPredicate) annotation.Annotations
}

type dependencyAnnotation struct {
	level AnnotationLevel
	ann   annotation.Annotation
}

type scopedDependencyAnnotationStore struct {
	annotations []dependencyAnnotation
}

var _ DependencyAnnotationStore = (*scopedDependencyAnnotationStore)(nil)

// NewScopedDependencyAnnotationStore creates a dependency-scoped annotation
// store for a single analyzer execution.
//
// The returned store includes only annotations emitted by the analyzers listed
// in dependencyAnalyzerIDs and supports level-aware filtering.
func NewScopedDependencyAnnotationStore(
	dependencyAnalyzerIDs []string,
	annotationsByLevel map[AnnotationLevel]annotation.Annotations,
) DependencyAnnotationStore {
	dependencySet := make(map[string]struct{}, len(dependencyAnalyzerIDs))
	for _, id := range dependencyAnalyzerIDs {
		if id == "" {
			continue
		}
		dependencySet[id] = struct{}{}
	}

	entries := make([]dependencyAnnotation, 0)
	for _, level := range []AnnotationLevel{
		AnnotationLevelProposal,
		AnnotationLevelBatchOperation,
		AnnotationLevelCall,
		AnnotationLevelParameter,
	} {
		for _, ann := range annotationsByLevel[level] {
			if ann == nil {
				continue
			}
			if _, ok := dependencySet[ann.AnalyzerID()]; !ok {
				continue
			}
			entries = append(entries, dependencyAnnotation{
				level: level,
				ann:   ann,
			})
		}
	}

	return &scopedDependencyAnnotationStore{annotations: entries}
}

func (s *scopedDependencyAnnotationStore) DependencyAnnotations() annotation.Annotations {
	result := make(annotation.Annotations, 0, len(s.annotations))
	for _, entry := range s.annotations {
		result = append(result, entry.ann)
	}

	return result
}

func (s *scopedDependencyAnnotationStore) Filter(preds ...DependencyAnnotationPredicate) annotation.Annotations {
	result := make(annotation.Annotations, 0)
	for _, entry := range s.annotations {
		matches := true
		for _, pred := range preds {
			if pred == nil {
				continue
			}
			if !pred(entry.level, entry.ann) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		result = append(result, entry.ann)
	}

	return result
}
