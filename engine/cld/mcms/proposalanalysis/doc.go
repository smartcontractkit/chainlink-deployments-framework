/*
Package proposalanalysis provides an engine for analyzing MCMS timelock proposals
and rendering the analyzed output.

# Overview

The engine workflow is:

 1. Create an engine with optional configuration.
 2. Register one or more analyzers.
 3. Register a renderer.
 4. Run analysis on a proposal.
 5. Render the analyzed proposal with RenderTo.

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
		req proposalanalysis.ProposalAnalyzeRequest,
		proposal proposalanalysis.DecodedTimelockProposal,
	) bool {
		_ = ctx
		_ = req
		_ = proposal
		return true
	}

	func (RiskAnalyzer) Analyze(
		ctx context.Context,
		req proposalanalysis.ProposalAnalyzeRequest,
		proposal proposalanalysis.DecodedTimelockProposal,
	) (proposalanalysis.Annotations, error) {
		_ = ctx
		_ = req
		_ = proposal

		return proposalanalysis.Annotations{
			proposalanalysis.NewAnnotation("risk-level", "string", "medium"),
		}, nil
	}

# Implementing a Renderer

A renderer must implement:

  - ID() string: unique renderer identifier.
  - RenderTo(io.Writer, RenderRequest, AnalyzedProposal) error

Example custom plain-text renderer:

	import (
		"fmt"
		"io"

		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
	)

	type PlainRenderer struct{}

	func (PlainRenderer) ID() string {
		return "plain"
	}

	func (PlainRenderer) RenderTo(
		w io.Writer,
		req proposalanalysis.RenderRequest,
		proposal proposalanalysis.AnalyzedProposal,
	) error {
		if _, err := fmt.Fprintf(
			w,
			"domain=%s env=%s batches=%d\n",
			req.Domain,
			req.EnvironmentName,
			len(proposal.BatchOperations()),
		); err != nil {
			return err
		}

		for _, batch := range proposal.BatchOperations() {
			if _, err := fmt.Fprintf(
				w,
				"- chain=%d calls=%d\n",
				batch.ChainSelector(),
				len(batch.Calls()),
			); err != nil {
				return err
			}
		}

		return nil
	}

# Usage

	import (
		"context"
		"os"
		"time"

		"github.com/smartcontractkit/mcms"

		"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
		cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
	)

	func Example() error {
		ctx := context.Background()

		engine := proposalanalysis.NewAnalyzerEngine(
			proposalanalysis.WithAnalyzerTimeout(90*time.Second),
		)

		// Register your analyzers (implementations omitted for brevity).
		if err := engine.RegisterAnalyzer(newDependencyAnalyzer()); err != nil {
			return err
		}
		if err := engine.RegisterAnalyzer(newRiskAnalyzer()); err != nil {
			return err
		}

		mdRenderer, err := proposalanalysis.NewMarkdownRenderer()
		if err != nil {
			return err
		}
		if err := engine.RegisterRenderer(mdRenderer); err != nil {
			return err
		}

		domain := cldfdomain.NewDomain("/path/to/domains", "ccip")
		env := ...
		proposal := ...

		analyzed, err := engine.Run(ctx, proposalanalysis.RunRequest{
			Domain:        domain,
			Environment:   env,
			DecoderConfig: proposalanalysis.DecoderConfig{},
		}, proposal)
		if err != nil {
			return err
		}

		return engine.RenderTo(
			os.Stdout,
			proposalanalysis.RendererIDMarkdown,
			proposalanalysis.RenderRequest{
				Domain:          domain.Key(),
				EnvironmentName: env.Name,
			},
			analyzed,
		)
	}
*/
package proposalanalysis
