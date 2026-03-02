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

# Usage

	import (
		"context"
		"os"
		"time"

		"github.com/smartcontractkit/mcms"

		"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
		cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
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

		mdRenderer, err := renderer.NewMarkdownRenderer()
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
			DecoderConfig: decoder.Config{},
		}, proposal)
		if err != nil {
			return err
		}

		return engine.RenderTo(
			os.Stdout,
			renderer.IDMarkdown,
			renderer.RenderRequest{
				Domain:          domain.Key(),
				EnvironmentName: env.Name,
			},
			analyzed,
		)
	}
*/
package proposalanalysis
