package analyzer

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	solana "github.com/gagliardetto/solana-go"
	"github.com/google/go-cmp/cmp"

	timelockbindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/timelock"

	datastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"

	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	chainsel "github.com/smartcontractkit/chain-selectors"
	cpistub "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/external_program_cpi_stub"
	"github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/mcm"

	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

func Test_solanaAnalyzer_describeOperations(t *testing.T) {
	t.Parallel()

	cpistub.SetProgramID(solana.MPK(cpiStubProgramID))
	mcm.SetProgramID(solana.MPK(mcmProgramID))
	solanaChainSelector := mcmstypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector)
	ds := datastore.NewMemoryDataStore()
	require.NoError(t, ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chainsel.SOLANA_DEVNET.Selector,
		Address:       cpiStubProgramID,
		Type:          "ExternalProgramCpiStub",
		Version:       &Version1_0_0,
	}))
	require.NoError(t, ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chainsel.SOLANA_DEVNET.Selector,
		Address:       mcmProgramID,
		Type:          "ManyChainMultiSigProgram",
		Version:       &Version1_0_0,
	}))
	env := deployment.Environment{DataStore: ds.Seal(), ExistingAddresses: deployment.NewMemoryAddressBook()}
	defaultProposalCtx, err := NewDefaultProposalContext(
		env,
		WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0": RBACTimelockMetaDataTesting.ABI,
		}),
		WithSolanaDecoders(map[string]DecodeInstructionFn{
			"ExternalProgramCpiStub 1.0.0":   DIFn(cpistub.DecodeInstruction),
			"ManyChainMultiSigProgram 1.0.0": DIFn(mcm.DecodeInstruction),
			"RBACTimelockProgram 1.0.0":      DIFn(timelockbindings.DecodeInstruction),
		}),
	)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ctx        ProposalContext
		operations []mcmstypes.Operation
		want       []string
		wantErr    string
	}{
		{
			name: "success: cpistub.Empty",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,

				Transaction: mcmsTxFromInstruction(t, cpistub.NewEmptyInstruction()),
			}},
			want: []string{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `Empty`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"[]\n" +
					"\n" + // <----- ADD THIS LINE
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: cpistub.U8InstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(123))),
			}},
			want: []string{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `U8InstructionData`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `Data` | See below: `Data` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>Data</summary>\n\n" +
					"```\n" +
					"123\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"[]\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: cpistub.StructInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewStructInstructionDataInstruction(cpistub.Value{Value: uint8(45)})),
			}},
			want: []string{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `StructInstructionData`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `Data` | See below: `Data` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>Data</summary>\n\n" +
					"```\n" +
					"\n" +
					"  value: 45\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"[]\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: cpistub.BigInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewBigInstructionDataInstruction([]byte{0x0, 0x1, 0x2, 0x3})),
			}},
			want: []string{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `BigInstructionData`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `Data` | See below: `Data` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>Data</summary>\n\n" +
					"```\n" +
					"0x00010203\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"[]\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: cpistub.AccountMut",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction: mcmsTxFromInstruction(t, cpistub.NewAccountMutInstruction(
					solana.MPK("H2qiK1CzW2DheLz9WAGSF1GbvLoqQv9hgS56Rk8Wh3uA"), // u8Value account
					solana.MPK("4cubrmdczDbRT8XyBwSR871meZU426S6xkiouzQpspVK"), // stub caller account
					solana.SystemProgramID,
				)),
			}},
			want: []string{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `AccountMut`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"\n" +
					"- H2qiK1CzW2DheLz9WAGSF1GbvLoqQv9hgS56Rk8Wh3uA   [writable]\n" +
					"- 4cubrmdczDbRT8XyBwSR871meZU426S6xkiouzQpspVK   [signer]\n" +
					"- 11111111111111111111111111111111\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: mcm.InitializeSignatures",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction: mcmsTxFromInstruction(t, mcm.NewInitSignaturesInstruction(
					[32]uint8{'m', 'c', 'm'},      // multisig id
					[32]uint8{'r', 'o', 'o', 't'}, // root
					uint32(1767225600),            // validUntil: 2026-Jan-01T00:00:00Z
					uint8(2),                      // totalSignatures
					solana.MPK("8UXavXj14P3khJyWSfeDeZ57YS7vo8ynkKemo2M2C1VU"), // signatures pda
					solana.MPK("J6fUzHuGEHmqpmmq1BMGfjfeYjPwg4TWsKsJB8WGihoJ"), // authority
					solana.SystemProgramID,
				)),
			}},
			want: []string{
				"**Address:** `Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j` <sub><i>address of ManyChainMultiSigProgram 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `InitSignatures`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `MultisigId` | See below: `MultisigId` |  |\n" +
					"| `Root` | See below: `Root` |  |\n" +
					"| `ValidUntil` | See below: `ValidUntil` |  |\n" +
					"| `TotalSignatures` | See below: `TotalSignatures` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>MultisigId</summary>\n\n" +
					"```\n" +
					"0x6d636d0000000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>Root</summary>\n\n" +
					"```\n" +
					"0x726f6f7400000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>ValidUntil</summary>\n\n" +
					"```\n" +
					"1767225600\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>TotalSignatures</summary>\n\n" +
					"```\n" +
					"2\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"\n" +
					"- 8UXavXj14P3khJyWSfeDeZ57YS7vo8ynkKemo2M2C1VU   [writable]\n" +
					"- J6fUzHuGEHmqpmmq1BMGfjfeYjPwg4TWsKsJB8WGihoJ   [writable] [signer]\n" +
					"- 11111111111111111111111111111111\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: mcm.SetRoot",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction: mcmsTxFromInstruction(t, mcm.NewSetRootInstruction(
					[32]uint8{'m', 'c', 'm'},      // multisig id
					[32]uint8{'r', 'o', 'o', 't'}, // root
					uint32(1767225600),            // validUntil: 2026-Jan-01T00:00:00Z
					mcm.RootMetadataInput{
						ChainId:              chainsel.SOLANA_DEVNET.Selector,
						Multisig:             solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd"),
						PreOpCount:           1,
						PostOpCount:          2,
						OverridePreviousRoot: true,
					},
					[][32]uint8{ // proof
						common.HexToHash("0x0000000000000000000000000000000000000001"),
						common.HexToHash("0x0000000000000000000000000000000000000002"),
					},
					solana.MPK("1EMwYGgmo3UPwmyUiPvCUPM5kdL52LHPJXSZNUN1pam"),  // signatures pda
					solana.MPK("AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd"), // root metadata pda
					solana.MPK("xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg"),  // seen signatures pda
					solana.MPK("FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3"), // expiring root and opcount pda
					solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd"), // config pda
					solana.MPK("Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7"), // authority
					solana.SystemProgramID,
				)),
			}},
			want: []string{
				"**Address:** `Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j` <sub><i>address of ManyChainMultiSigProgram 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `SetRoot`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `MultisigId` | See below: `MultisigId` |  |\n" +
					"| `Root` | See below: `Root` |  |\n" +
					"| `ValidUntil` | See below: `ValidUntil` |  |\n" +
					"| `Metadata` | See below: `Metadata` |  |\n" +
					"| `MetadataProof` | See below: `MetadataProof` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>MultisigId</summary>\n\n" +
					"```\n" +
					"0x6d636d0000000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>Root</summary>\n\n" +
					"```\n" +
					"0x726f6f7400000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>ValidUntil</summary>\n\n" +
					"```\n" +
					"1767225600\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>Metadata</summary>\n\n" +
					"```\n" +
					"\n" + // <-- extra blank line before struct lines!
					"  chainid: 16423721717087811551\n" +
					"  multisig: 7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd\n" +
					"  preopcount: 1\n" +
					"  postopcount: 2\n" +
					"  overridepreviousroot: true\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>MetadataProof</summary>\n\n" +
					"```\n" +
					"\n" + // <-- extra blank line before proof lines!
					"- 0x0000000000000000000000000000000000000000000000000000000000000001\n" +
					"- 0x0000000000000000000000000000000000000000000000000000000000000002\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"\n" + // <-- extra blank line before account lines!
					"- 1EMwYGgmo3UPwmyUiPvCUPM5kdL52LHPJXSZNUN1pam    [writable]\n" +
					"- AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd   [writable]\n" +
					"- xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg    [writable]\n" +
					"- FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3   [writable]\n" +
					"- 7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd\n" +
					"- Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7   [writable] [signer]\n" +
					"- 11111111111111111111111111111111\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
		{
			name: "success: mcm.SetConfig",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction: mcmsTxFromInstruction(t, mcm.NewSetConfigInstruction(
					[32]uint8{'m', 'c', 'm'}, // multisig id
					[]byte{1, 2},             // signer groups
					[32]uint8{3, 4, 5},       // group quorums
					[32]uint8{6, 7, 8},       // group parents
					true,                     // clear root
					solana.MPK("AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd"), // multisig config pda
					solana.MPK("xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg"),  // signers pda
					solana.MPK("FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3"), // root metadata pda
					solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd"), // expiring root and opcount pda
					solana.MPK("Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7"), // authority
					solana.SystemProgramID,
				)),
			}},
			want: []string{
				"**Address:** `Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j` <sub><i>address of ManyChainMultiSigProgram 1.0.0 from solana-devnet</i></sub>\n" +
					"**Method:** `SetConfig`\n\n" +
					"**Inputs:**\n\n" +
					"| Name | Value | Annotation |\n" +
					"|------|-------|------------|\n" +
					"| `MultisigId` | See below: `MultisigId` |  |\n" +
					"| `SignerGroups` | See below: `SignerGroups` |  |\n" +
					"| `GroupQuorums` | See below: `GroupQuorums` |  |\n" +
					"| `GroupParents` | See below: `GroupParents` |  |\n" +
					"| `ClearRoot` | See below: `ClearRoot` |  |\n" +
					"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
					"<details><summary>MultisigId</summary>\n\n" +
					"```\n" +
					"0x6d636d0000000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>SignerGroups</summary>\n\n" +
					"```\n" +
					"0x0102\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>GroupQuorums</summary>\n\n" +
					"```\n" +
					"0x0304050000000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>GroupParents</summary>\n\n" +
					"```\n" +
					"0x0607080000000000000000000000000000000000000000000000000000000000\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>ClearRoot</summary>\n\n" +
					"```\n" +
					"true\n" +
					"\n" +
					"```\n" +
					"</details>\n\n" +
					"<details><summary>AccountMetaSlice</summary>\n\n" +
					"```\n" +
					"\n" + // extra blank line before accounts!
					"- AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd   [writable]\n" +
					"- xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg    [writable]\n" +
					"- FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3   [writable]\n" +
					"- 7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd   [writable]\n" +
					"- Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7   [writable] [signer]\n" +
					"- 11111111111111111111111111111111\n" +
					"\n" +
					"```\n" +
					"</details>\n\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := describeOperations(tt.ctx, tt.operations)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_solanaAnalyzer_describeBatchOperations(t *testing.T) {
	t.Parallel()

	cpistub.SetProgramID(solana.MPK(cpiStubProgramID))
	solanaChainSelector := mcmstypes.ChainSelector(chainsel.SOLANA_DEVNET.Selector)
	ds := datastore.NewMemoryDataStore()
	err := ds.Addresses().Add(datastore.AddressRef{
		ChainSelector: chainsel.SOLANA_DEVNET.Selector,
		Address:       cpiStubProgramID,
		Type:          "ExternalProgramCpiStub",
		Version:       &Version1_0_0,
	})
	require.NoError(t, err)
	env := deployment.Environment{
		ExistingAddresses: deployment.NewMemoryAddressBook(),
		DataStore:         ds.Seal(),
	}

	proposalCtx, err := NewDefaultProposalContext(
		env,
		WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0": RBACTimelockMetaDataTesting.ABI,
		}),
		WithSolanaDecoders(map[string]DecodeInstructionFn{
			"RBACTimelockProgram 1.0.0": DIFn(timelockbindings.DecodeInstruction),

			deployment.NewTypeAndVersion("ExternalProgramCpiStub", Version1_0_0).String(): DIFn(cpistub.DecodeInstruction),
		}),
	)

	require.NoError(t, err)

	tests := []struct {
		name     string
		ctx      ProposalContext
		batchOps []mcmstypes.BatchOperation
		want     [][]string
		wantErr  string
	}{
		{
			name: "success: multiple calls to cpistub.U8InstructionData split into 2 batches",
			ctx:  proposalCtx,
			batchOps: []mcmstypes.BatchOperation{
				{
					ChainSelector: solanaChainSelector,
					Transactions: []mcmstypes.Transaction{
						mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(12))),
						mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(34))),
					},
				},
				{
					ChainSelector: solanaChainSelector,
					Transactions: []mcmstypes.Transaction{
						mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(56))),
						mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(78))),
					},
				},
			},
			want: [][]string{
				{
					"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
						"**Method:** `U8InstructionData`\n\n" +
						"**Inputs:**\n\n" +
						"| Name | Value | Annotation |\n" +
						"|------|-------|------------|\n" +
						"| `Data` | See below: `Data` |  |\n" +
						"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
						"<details><summary>Data</summary>\n\n" +
						"```\n" +
						"12\n" +
						"\n" +
						"```\n" +
						"</details>\n\n" +
						"<details><summary>AccountMetaSlice</summary>\n\n" +
						"```\n" +
						"[]\n" +
						"\n" +
						"```\n" +
						"</details>\n\n",

					"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
						"**Method:** `U8InstructionData`\n\n" +
						"**Inputs:**\n\n" +
						"| Name | Value | Annotation |\n" +
						"|------|-------|------------|\n" +
						"| `Data` | See below: `Data` |  |\n" +
						"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
						"<details><summary>Data</summary>\n\n" +
						"```\n" +
						"34\n" +
						"\n" +
						"```\n" +
						"</details>\n\n" +
						"<details><summary>AccountMetaSlice</summary>\n\n" +
						"```\n" +
						"[]\n" +
						"\n" +
						"```\n" +
						"</details>\n\n",
				},
				{
					"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
						"**Method:** `U8InstructionData`\n\n" +
						"**Inputs:**\n\n" +
						"| Name | Value | Annotation |\n" +
						"|------|-------|------------|\n" +
						"| `Data` | See below: `Data` |  |\n" +
						"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
						"<details><summary>Data</summary>\n\n" +
						"```\n" +
						"56\n" +
						"\n" +
						"```\n" +
						"</details>\n\n" +
						"<details><summary>AccountMetaSlice</summary>\n\n" +
						"```\n" +
						"[]\n" +
						"\n" +
						"```\n" +
						"</details>\n\n",

					"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm` <sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>\n" +
						"**Method:** `U8InstructionData`\n\n" +
						"**Inputs:**\n\n" +
						"| Name | Value | Annotation |\n" +
						"|------|-------|------------|\n" +
						"| `Data` | See below: `Data` |  |\n" +
						"| `AccountMetaSlice` | See below: `AccountMetaSlice` |  |\n\n" +
						"<details><summary>Data</summary>\n\n" +
						"```\n" +
						"78\n" +
						"\n" +
						"```\n" +
						"</details>\n\n" +
						"<details><summary>AccountMetaSlice</summary>\n\n" +
						"```\n" +
						"[]\n" +
						"\n" +
						"```\n" +
						"</details>\n\n",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := describeBatchOperations(tt.ctx, tt.batchOps)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func mcmsTxFromInstruction[T any](t *testing.T, instructionBuilder interface{ ValidateAndBuild() (T, error) }) mcmstypes.Transaction {
	t.Helper()

	var instruction any
	var err error

	instruction, err = instructionBuilder.ValidateAndBuild()
	require.NoError(t, err)
	solanaInstruction, ok := instruction.(solana.Instruction)
	require.True(t, ok)
	tx, err := mcmssolanasdk.NewTransactionFromInstruction(solanaInstruction, "ExternalProgramCpiStub", nil)
	require.NoError(t, err)

	return tx
}

const (
	cpiStubProgramID = "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm"
	mcmProgramID     = "Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j"
)
