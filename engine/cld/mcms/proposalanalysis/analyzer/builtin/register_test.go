package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
)

func TestRegisterAll(t *testing.T) {
	t.Parallel()

	registry := analyzer.NewRegistry()
	RegisterAll(registry)

	require.Empty(t, registry.All())
}
