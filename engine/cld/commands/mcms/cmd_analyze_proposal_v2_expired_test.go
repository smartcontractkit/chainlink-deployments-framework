package mcms

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func TestAnalyzeProposalV2_validUntil(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		validUntil string
	}{
		{name: "not expired", validUntil: `"validUntil": 2004259681`},
		{name: "expired", validUntil: `"validUntil": 1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// A native transfer decodes without an ABI registry.
			proposalJSON := strings.NewReplacer(
				`"validUntil": 2004259681`, tt.validUntil,
				`"additionalFields": {"value": 0}`, `"additionalFields": {"value": 1}`,
				`"data": "ZGF0YQ=="`, `"data": ""`,
			).Replace(string(testProposalWithoutChangesetsJSON))
			proposalPath := filepath.Join(t.TempDir(), "proposal.json")
			require.NoError(t, os.WriteFile(proposalPath, []byte(proposalJSON), 0o600))

			cmd := newAnalyzeProposalV2Cmd(Config{
				Logger: logger.Nop(),
				Domain: domain.NewDomain(t.TempDir(), "test"),
				ProposalContextProvider: func(cldf.Environment) (analyzer.ProposalContext, error) {
					return &analyzer.DefaultProposalContext{AddressesByChain: cldf.AddressesByChain{}}, nil
				},
				Deps: Deps{EnvironmentLoader: func(
					_ context.Context, _ domain.Domain, envKey string, _ logger.Logger, _ ...cldfenv.LoadEnvironmentOption,
				) (cldf.Environment, error) {
					return cldf.Environment{
						Name:              envKey,
						ExistingAddresses: cldf.NewMemoryAddressBook(), //nolint:staticcheck // SA1019 (Deprecated): We still need to support AddressBook for now
						DataStore:         datastore.NewMemoryDataStore().Seal(),
					}, nil
				}},
			})
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"-e", "testnet", "-p", proposalPath})

			require.NoError(t, cmd.ExecuteContext(t.Context()))
			require.Contains(t, out.String(), "`native_transfer`")
		})
	}
}
