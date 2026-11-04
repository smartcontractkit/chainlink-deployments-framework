package simulation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

func TestLoadArtifact(t *testing.T) {
	t.Parallel()

	artifact := &SimulationArtifact{
		SchemaVersion:       ArtifactSchemaVersion,
		ProposalSigningHash: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		GeneratedAt:         time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
		Chains: map[string]SimulationChainResult{
			"5009297550715157269": {
				ForkBlockNumber: 25784394,
				Simulated:       true,
				Changes: []statediff.Change{{
					ChainSelector: 5009297550715157269,
					Address:       "0xAbC",
					ContractType:  "Router",
					Path:          "chains.5009297550715157269.0xAbC_Router.onRamps",
					Kind:          statediff.KindAdded,
					After:         "0xdef",
				}},
				Unreadable: []statediff.Unreadable{},
				Assertions: []Assertion{},
			},
		},
	}
	data, err := json.MarshalIndent(artifact, "", "  ")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "sim.statediff.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	loaded, err := LoadArtifact(path)
	require.NoError(t, err)
	require.Equal(t, artifact.SchemaVersion, loaded.SchemaVersion)
	require.Equal(t, artifact.ProposalSigningHash, loaded.ProposalSigningHash)
	require.True(t, loaded.GeneratedAt.Equal(artifact.GeneratedAt))
	require.Equal(t, artifact.Chains, loaded.Chains)

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		_, err := LoadArtifact(filepath.Join(t.TempDir(), "nope.json"))
		require.ErrorContains(t, err, "reading simulation artifact")
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "bad.json")
		require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))

		_, err := LoadArtifact(path)
		require.ErrorContains(t, err, "parsing simulation artifact")
	})

	t.Run("wrong schema version", func(t *testing.T) {
		t.Parallel()

		future := &SimulationArtifact{SchemaVersion: ArtifactSchemaVersion + 1, Chains: map[string]SimulationChainResult{}}
		data, err := json.Marshal(future)
		require.NoError(t, err)
		path := filepath.Join(t.TempDir(), "future.json")
		require.NoError(t, os.WriteFile(path, data, 0o600))

		_, err = LoadArtifact(path)
		require.ErrorContains(t, err, "schema version")
	})
}

func TestLoadArtifactForSigningHash(t *testing.T) {
	t.Parallel()

	write := func(t *testing.T, hash string) string {
		t.Helper()

		artifact := &SimulationArtifact{
			SchemaVersion:       ArtifactSchemaVersion,
			ProposalSigningHash: hash,
			Chains:              map[string]SimulationChainResult{},
		}
		data, err := json.Marshal(artifact)
		require.NoError(t, err)
		path := filepath.Join(t.TempDir(), "sim.statediff.json")
		require.NoError(t, os.WriteFile(path, data, 0o600))

		return path
	}

	t.Run("matching hash loads", func(t *testing.T) {
		t.Parallel()

		loaded, err := LoadArtifactForSigningHash(write(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		require.NoError(t, err)
		require.Equal(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", loaded.ProposalSigningHash)
	})

	t.Run("mismatching hash is stale", func(t *testing.T) {
		t.Parallel()

		_, err := LoadArtifactForSigningHash(write(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.ErrorContains(t, err, "stale")
		require.ErrorContains(t, err, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		require.ErrorContains(t, err, "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	})
}

func TestMergeArtifacts(t *testing.T) {
	t.Parallel()

	ethResult := SimulationChainResult{
		ForkBlockNumber: 25784394,
		Simulated:       true,
		Changes:         []statediff.Change{{ChainSelector: 5009297550715157269, Address: "0xabc", Path: "chains.5009297550715157269.0xabc_Router.onRamps", Kind: statediff.KindAdded, After: "0xdef"}},
		Unreadable:      []statediff.Unreadable{},
		Assertions:      []Assertion{},
	}
	rskNotSimulated := SimulationChainResult{
		Simulated:  false,
		Reason:     "non-EVM chain",
		Changes:    []statediff.Change{},
		Unreadable: []statediff.Unreadable{},
		Assertions: []Assertion{},
	}
	rskNotSimulatedLater := rskNotSimulated
	rskNotSimulatedLater.Reason = "bypassed fork test"

	newArtifact := func(hash string, generatedAt time.Time, chains map[string]SimulationChainResult) *SimulationArtifact {
		return &SimulationArtifact{
			SchemaVersion:       ArtifactSchemaVersion,
			ProposalSigningHash: hash,
			GeneratedAt:         generatedAt,
			Chains:              chains,
		}
	}

	t.Run("merges chains from several artifacts", func(t *testing.T) {
		t.Parallel()

		eth := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"5009297550715157269": ethResult})
		rsk := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 11, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"11964252391146578476": rskNotSimulated})

		merged, err := MergeArtifacts(eth, rsk)
		require.NoError(t, err)
		require.Equal(t, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", merged.ProposalSigningHash)
		require.Len(t, merged.Chains, 2)
		require.Equal(t, ethResult, merged.Chains["5009297550715157269"])
		require.Equal(t, rskNotSimulated, merged.Chains["11964252391146578476"])
		// generatedAt is the latest of the inputs.
		require.True(t, merged.GeneratedAt.Equal(rsk.GeneratedAt))
	})

	t.Run("single artifact passes through", func(t *testing.T) {
		t.Parallel()

		eth := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"5009297550715157269": ethResult})

		merged, err := MergeArtifacts(eth)
		require.NoError(t, err)
		require.Equal(t, eth.Chains, merged.Chains)
	})

	t.Run("duplicate identical chain tolerated", func(t *testing.T) {
		t.Parallel()

		first := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"5009297550715157269": ethResult})
		second := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"5009297550715157269": ethResult})

		merged, err := MergeArtifacts(first, second)
		require.NoError(t, err)
		require.Len(t, merged.Chains, 1)
	})

	t.Run("conflicting duplicate chain errors", func(t *testing.T) {
		t.Parallel()

		first := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"11964252391146578476": rskNotSimulated})
		second := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 11, 0, 0, 0, time.UTC),
			map[string]SimulationChainResult{"11964252391146578476": rskNotSimulatedLater})

		_, err := MergeArtifacts(first, second)
		require.ErrorContains(t, err, "conflicting simulation results for chain 11964252391146578476")
	})

	t.Run("signing hash mismatch errors", func(t *testing.T) {
		t.Parallel()

		first := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), map[string]SimulationChainResult{})
		second := newArtifact("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", time.Date(2026, 9, 12, 11, 0, 0, 0, time.UTC), map[string]SimulationChainResult{})

		_, err := MergeArtifacts(first, second)
		require.ErrorContains(t, err, "different proposals cannot be merged")
	})

	t.Run("schema version mismatch errors", func(t *testing.T) {
		t.Parallel()

		first := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), map[string]SimulationChainResult{})
		second := newArtifact("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", time.Date(2026, 9, 12, 11, 0, 0, 0, time.UTC), map[string]SimulationChainResult{})
		second.SchemaVersion = ArtifactSchemaVersion + 1

		_, err := MergeArtifacts(first, second)
		require.ErrorContains(t, err, "schema version")
	})

	t.Run("no artifacts errors", func(t *testing.T) {
		t.Parallel()

		_, err := MergeArtifacts()
		require.ErrorContains(t, err, "no simulation artifacts to merge")
	})
}
