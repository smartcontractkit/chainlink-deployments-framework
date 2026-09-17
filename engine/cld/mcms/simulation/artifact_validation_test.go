package simulation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

func validationArtifact() *SimulationArtifact {
	return &SimulationArtifact{SchemaVersion: ArtifactSchemaVersion, ProposalSigningHash: "0x" + strings.Repeat("a", 64), Chains: map[string]SimulationChainResult{
		"1": {Simulated: true, Targets: []Target{{Address: "0xabc", ContractType: "Router"}}, Changes: []statediff.Change{{ChainSelector: 1, Address: "0xabc", Path: "chains.1.0xabc_Router.value", Kind: statediff.KindChanged, Before: json.Number("9007199254740992"), After: json.Number("9007199254740993")}}},
	}}
}

func TestArtifactValidationRejectsInconsistentResults(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		mutate func(*SimulationArtifact)
		want   string
	}{
		{"failed changes", func(a *SimulationArtifact) {
			c := a.Chains["1"]
			c.Simulated = false
			c.Reason = "failed"
			a.Chains["1"] = c
		}, "not simulated but carries changes"},
		{"wrong selector", func(a *SimulationArtifact) { a.Chains["1"].Changes[0].ChainSelector = 2 }, "invalid change"},
		{"wrong path selector", func(a *SimulationArtifact) { a.Chains["1"].Changes[0].Path = "chains.2.0xabc_Router.value" }, "path does not identify chain"},
		{"wrong address", func(a *SimulationArtifact) { a.Chains["1"].Changes[0].Address = "0xdef" }, "path/address mismatch"},
		{"unreadable with changes", func(a *SimulationArtifact) {
			c := a.Chains["1"]
			c.Unreadable = []statediff.Unreadable{{ChainSelector: 1, Address: "0xABC", Reason: "read failed"}}
			a.Chains["1"] = c
		}, "unreadable target"},
		{"missing reason", func(a *SimulationArtifact) { a.Chains["1"] = SimulationChainResult{} }, "has no reason"},
		{"bad selector", func(a *SimulationArtifact) { a.Chains["01"] = a.Chains["1"] }, "invalid chain selector"},
		{"bad hash", func(a *SimulationArtifact) { a.ProposalSigningHash = "0xabc" }, "32-byte hash"},
		{"outside inventory", func(a *SimulationArtifact) { c := a.Chains["1"]; c.Targets = []Target{}; a.Chains["1"] = c }, "outside target inventory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := validationArtifact()
			tc.mutate(a)
			require.ErrorContains(t, a.Validate(), tc.want)
		})
	}
}

func TestArtifactExactNumbersAndExpectedChains(t *testing.T) {
	t.Parallel()
	a := validationArtifact()
	raw, err := json.Marshal(a)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "exact.json")
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	loaded, err := LoadArtifact(path)
	require.NoError(t, err)
	require.Equal(t, json.Number("9007199254740993"), loaded.Chains["1"].Changes[0].After)
	equivalent := validationArtifact()
	equivalent.Chains["1"].Changes[0].After = json.Number("9007199254740993.0")
	_, err = MergeArtifacts(loaded, equivalent)
	require.NoError(t, err)
	require.NoError(t, loaded.ValidateExpectedChains([]string{"1"}))
	require.ErrorContains(t, loaded.ValidateExpectedChains([]string{"1", "2"}), "missing simulation result")
	require.ErrorContains(t, loaded.ValidateExpectedChains([]string{"1", "1"}), "duplicate")
	require.ErrorContains(t, loaded.ValidateExpectedChains(nil), "must not be empty")
	loaded.Chains["2"] = SimulationChainResult{Reason: "unsupported"}
	require.ErrorContains(t, loaded.ValidateExpectedChains([]string{"1"}), "unexpected simulation result")
	require.NoError(t, os.WriteFile(path, append(raw, []byte(" {}")...), 0o600))
	_, err = LoadArtifact(path)
	require.ErrorContains(t, err, "parsing simulation artifact")
}
