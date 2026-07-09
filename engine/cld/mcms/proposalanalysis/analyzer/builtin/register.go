package builtin

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/builtin/timelockdelay"
)

// RegisterAll registers built-in proposal analyzers with the registry.
func RegisterAll(registry *analyzer.Registry) {
	register := func(a analyzer.BaseAnalyzer) {
		if err := registry.Register(a); err != nil {
			panic(fmt.Sprintf("proposalanalysis: register built-in analyzer %q: %v", a.ID(), err))
		}
	}

	register(timelockdelay.Validator{})
}
