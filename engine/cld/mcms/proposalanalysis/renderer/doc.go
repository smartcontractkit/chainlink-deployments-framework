/*
Package renderer defines renderer interfaces used to format analyzed proposals.

# Implementing a Renderer

A renderer must implement:

  - ID() string: unique renderer identifier.
  - RenderTo(io.Writer, RenderRequest, analyzer.AnalyzedProposal) error

Example custom plain-text renderer:

	import (
		"fmt"
		"io"

		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
		"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
	)

	type PlainRenderer struct{}

	func (PlainRenderer) ID() string {
		return "plain"
	}

	func (PlainRenderer) RenderTo(
		w io.Writer,
		req renderer.RenderRequest,
		proposal analyzer.AnalyzedProposal,
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

# Registering with the engine

	eng := proposalanalysis.NewAnalyzerEngine()
	if err := eng.RegisterRenderer(PlainRenderer{}); err != nil {
		return err
	}

To render output, call `eng.RenderTo(...)` with the renderer ID and a
`renderer.RenderRequest`.
*/
package renderer
