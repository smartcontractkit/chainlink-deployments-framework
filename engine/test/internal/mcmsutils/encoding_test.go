package mcmsutils

import (
	"os"
	"path/filepath"
	"testing"

	mcmslib "github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeProposal(t *testing.T) {
	t.Parallel()

	// Decode a proposal from test data
	proposalJSON, err := os.ReadFile(filepath.Join("testdata", "proposal.json"))
	require.NoError(t, err, "Failed to read test data file")

	proposal, err := DecodeProposal(string(proposalJSON))
	require.NoError(t, err, "Failed to decode original proposal")

	t.Run("successful encoding", func(t *testing.T) {
		t.Parallel()

		encodedJSON, err := EncodeProposal(proposal)
		require.NoError(t, err)
		assert.NotEmpty(t, encodedJSON)

		assert.JSONEq(t, string(proposalJSON), encodedJSON)
	})
}

func TestDecodeProposal(t *testing.T) {
	t.Parallel()

	// Read the test data file
	proposalJSON, rerr := os.ReadFile(filepath.Join("testdata", "proposal.json"))
	require.NoError(t, rerr, "Failed to read test data file")

	t.Run("successful decoding", func(t *testing.T) {
		t.Parallel()

		var proposal *mcmslib.Proposal
		proposal, err := DecodeProposal(string(proposalJSON))
		require.NoError(t, err)
		require.NotNil(t, proposal)

		// Basic assertions since the we use mcmslib to decode the proposal
		assert.Equal(t, "v1", proposal.Version)
		assert.Equal(t, mcmstypes.ProposalKind("Proposal"), proposal.Kind)
		require.Contains(t, proposal.ChainMetadata, mcmstypes.ChainSelector(3379446385462418246))
		require.Len(t, proposal.Operations, 1)
	})

	t.Run("failed decoding", func(t *testing.T) {
		t.Parallel()

		_, err := DecodeProposal("invalid JSON")
		require.Error(t, err)
	})
}

func TestEncodeTimelockProposal(t *testing.T) {
	t.Parallel()

	// Decode a timelock proposal from test data
	timelockJSON, err := os.ReadFile(filepath.Join("testdata", "timelock_proposal.json"))
	require.NoError(t, err, "Failed to read timelock test data file")

	timelockProposal, err := DecodeTimelockProposal(string(timelockJSON))
	require.NoError(t, err, "Failed to decode original timelock proposal")

	t.Run("successful encoding", func(t *testing.T) {
		t.Parallel()

		encodedJSON, err := EncodeTimelockProposal(timelockProposal)
		require.NoError(t, err)
		assert.NotEmpty(t, encodedJSON)

		// Verify we can decode it back successfully (instead of strict JSON comparison)
		decodedProposal, err := DecodeTimelockProposal(encodedJSON)
		require.NoError(t, err)
		require.NotNil(t, decodedProposal)

		assert.JSONEq(t, string(timelockJSON), encodedJSON)
	})
}

func TestDecodeTimelockProposal(t *testing.T) {
	t.Parallel()

	// Read the timelock test data file
	timelockJSON, rerr := os.ReadFile(filepath.Join("testdata", "timelock_proposal.json"))
	require.NoError(t, rerr, "Failed to read timelock test data file")

	t.Run("successful decoding", func(t *testing.T) {
		t.Parallel()

		var timelockProposal *mcmslib.TimelockProposal
		timelockProposal, err := DecodeTimelockProposal(string(timelockJSON))
		require.NoError(t, err)
		require.NotNil(t, timelockProposal)

		// Basic assertions since the we use mcmslib to decode the proposal
		assert.Equal(t, "v1", timelockProposal.Version)
		assert.Equal(t, mcmstypes.ProposalKind("TimelockProposal"), timelockProposal.Kind)
		assert.Equal(t, mcmstypes.TimelockActionSchedule, timelockProposal.Action)
		assert.Contains(t, timelockProposal.ChainMetadata, mcmstypes.ChainSelector(16015286601757825753))
		assert.Len(t, timelockProposal.Operations, 1)
	})

	t.Run("failed decoding", func(t *testing.T) {
		t.Parallel()

		_, err := DecodeTimelockProposal("invalid JSON")
		require.Error(t, err)
	})
}
