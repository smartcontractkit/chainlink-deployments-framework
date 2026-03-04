package mcms

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/mcms"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/renderer"
)

func TestRegisterProposalRenderers(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		engine := &testAnalyzerEngine{}
		renderers := []renderer.Renderer{
			testRenderer{id: "plain"},
			testRenderer{id: "html"},
		}

		err := registerProposalRenderers(engine, renderers)

		require.NoError(t, err)
		require.Equal(t, []string{"plain", "html"}, engine.registeredIDs)
	})

	t.Run("registration error includes renderer id", func(t *testing.T) {
		t.Parallel()

		engine := &testAnalyzerEngine{registerRendererErr: errors.New("boom")}
		renderers := []renderer.Renderer{
			testRenderer{id: "custom"},
		}

		err := registerProposalRenderers(engine, renderers)

		require.EqualError(t, err, `register proposal renderer "custom": boom`)
	})
}

type testRenderer struct {
	id string
}

func (t testRenderer) ID() string {
	return t.id
}

func (t testRenderer) RenderTo(_ io.Writer, _ renderer.RenderRequest, _ analyzer.AnalyzedProposal) error {
	return nil
}

type testAnalyzerEngine struct {
	registerRendererErr error
	registeredIDs       []string
}

func (t *testAnalyzerEngine) Run(_ context.Context, _ proposalanalysis.RunRequest, _ *mcms.TimelockProposal) (analyzer.AnalyzedProposal, error) {
	return nil, nil //nolint:nilnil
}

func (t *testAnalyzerEngine) RegisterAnalyzer(_ analyzer.BaseAnalyzer) error {
	return nil
}

func (t *testAnalyzerEngine) RegisterRenderer(r renderer.Renderer) error {
	if t.registerRendererErr != nil {
		return t.registerRendererErr
	}

	t.registeredIDs = append(t.registeredIDs, r.ID())

	return nil
}

func (t *testAnalyzerEngine) RenderTo(_ io.Writer, _ string, _ renderer.RenderRequest, _ analyzer.AnalyzedProposal) error {
	return nil
}
