package builtin

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/builtin/timelockdelay"
)

func TestRegisterAll(t *testing.T) {
	t.Parallel()

	registry := analyzer.NewRegistry()
	RegisterAll(registry)

	_, ok := registry.Get(timelockdelay.ValidatorID)
	require.True(t, ok)
}
