/*
Package analyzer defines analyzer interfaces used by the proposal analysis engine.

# Implementing an Analyzer

Every analyzer must implement `BaseAnalyzer`:

  - ID() string: unique identifier.
  - Dependencies() []string: analyzer IDs that must run first.

Then implement one of the scope-specific analyzer interfaces:

  - ProposalAnalyzer
  - BatchOperationAnalyzer
  - CallAnalyzer
  - ParameterAnalyzer

Example proposal-level analyzer:

	import (
		"context"

		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	)

	type RiskAnalyzer struct{}

	func (RiskAnalyzer) ID() string {
		return "risk-analyzer"
	}

	func (RiskAnalyzer) Dependencies() []string {
		return []string{"dependency-analyzer"}
	}

	func (RiskAnalyzer) CanAnalyze(
		ctx context.Context,
		req analyzer.ProposalAnalyzeRequest,
		proposal decoder.DecodedTimelockProposal,
	) bool {
		_ = ctx
		_ = req
		_ = proposal
		return true
	}

	func (RiskAnalyzer) Analyze(
		ctx context.Context,
		req analyzer.ProposalAnalyzeRequest,
		proposal decoder.DecodedTimelockProposal,
	) (annotation.Annotations, error) {
		_ = ctx
		_ = req
		_ = proposal

		return annotation.Annotations{
			annotation.New("risk-level", "string", "medium"),
		}, nil
	}

# Registering with the engine

	eng := proposalanalysis.NewAnalyzerEngine()
	if err := eng.RegisterAnalyzer(RiskAnalyzer{}); err != nil {
		return err
	}

The engine resolves analyzer execution order using `Dependencies()` and passes
dependency-scoped annotations through the request's `DependencyAnnotationStore`.
*/
package analyzer
