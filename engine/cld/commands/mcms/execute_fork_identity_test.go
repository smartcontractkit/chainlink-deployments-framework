package mcms

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/chainwrappers"
	"github.com/smartcontractkit/mcms/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestRunExecuteForkCapturesAuthoredHashBeforeSaltAndStartup(t *testing.T) {
	t.Parallel()
	var proposal mcms.TimelockProposal
	require.NoError(t, json.Unmarshal(testProposalWithoutChangesetsJSON, &proposal))
	converters, err := chainwrappers.BuildConverters(proposal.ChainMetadata)
	require.NoError(t, err)
	converted, _, err := proposal.Convert(t.Context(), converters)
	require.NoError(t, err)
	authoredHash, err := converted.SigningHash()
	require.NoError(t, err)
	originalSalt := proposal.SaltOverride
	output := filepath.Join(t.TempDir(), "simulation.statediff.json")
	require.NoError(t, os.WriteFile(output, []byte("stale artifact from prior run"), 0o600))

	loads := 0
	cfg := Config{Logger: logger.Nop(), Deps: Deps{
		ProposalLoader: func(_ types.ProposalKind, _ string) (mcms.ProposalInterface, error) {
			loads++
			return &proposal, nil
		},
		ForkEnvironmentLoader: func(_ context.Context, _ domain.Domain, _ string, _ map[uint64]*big.Int, _ ...environment.LoadEnvironmentOption) (environment.ForkedEnvironment, error) {
			return environment.ForkedEnvironment{}, errors.New("test fork startup failed")
		},
	}}
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	err = runExecuteFork(cmd, cfg, executeForkFlags{
		proposalKind: string(types.KindTimelockProposal), chainSelector: chainsel.GETH_TESTNET.Selector,
		randomSalt: true, simulateState: true, simulationOut: output,
	})
	require.ErrorContains(t, err, "test fork startup failed")
	require.Equal(t, 1, loads, "the authored proposal must be loaded exactly once")
	require.NotEqual(t, originalSalt, proposal.SaltOverride)
	mutated, _, err := proposal.Convert(t.Context(), converters)
	require.NoError(t, err)
	mutatedHash, err := mutated.SigningHash()
	require.NoError(t, err)
	require.NotEqual(t, authoredHash, mutatedHash)

	artifact, err := simulation.LoadArtifact(output)
	require.NoError(t, err)
	require.Equal(t, authoredHash.Hex(), artifact.ProposalSigningHash)
	chain, present := artifact.Chains[strconv.FormatUint(chainsel.GETH_TESTNET.Selector, 10)]
	require.True(t, present)
	require.False(t, chain.Simulated)
	require.Contains(t, chain.Reason, "test fork startup failed")
}
