package analyzer

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	solana "github.com/gagliardetto/solana-go"

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

//nolint:paralleltest // call to SetProgramID is not thread-safe
func Test_solanaAnalyzer_describeOperations(t *testing.T) {
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
		name         string
		ctx          ProposalContext
		operations   []mcmstypes.Operation
		wantContains [][]string // per operation, substrings that must be present
		wantErr      string
	}{
		{
			name: "success: cpistub.Empty",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,

				Transaction: mcmsTxFromInstruction(t, cpistub.NewEmptyInstruction()),
			}},
			wantContains: [][]string{{
				"**Address:** `2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm`",
				"<sub><i>address of ExternalProgramCpiStub 1.0.0 from solana-devnet</i></sub>",
				"**Method:** `Empty`",
				"**Inputs:**\n\n",
				"- `AccountMetaSlice`:",
				"<details><summary>AccountMetaSlice</summary>",
				"[]",
			}},
		},
		{
			name: "success: cpistub.U8InstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(123))),
			}},
			wantContains: [][]string{{
				"**Method:** `U8InstructionData`",
				"- `Data`: 123",
				"<details><summary>Data</summary>",
				"12",
				"- `AccountMetaSlice`:",
				"<details><summary>AccountMetaSlice</summary>",
				"[]",
			}},
		},
		{
			name: "success: cpistub.StructInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewStructInstructionDataInstruction(cpistub.Value{Value: uint8(45)})),
			}},
			wantContains: [][]string{{
				"**Method:** `StructInstructionData`",
				"- `Data`:",
				"<details><summary>Data</summary>",
				"value: 45",
				"- `AccountMetaSlice`:",
				"<details><summary>AccountMetaSlice</summary>",
				"[]",
			}},
		},
		{
			name: "success: cpistub.BigInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewBigInstructionDataInstruction([]byte{0x0, 0x1, 0x2, 0x3})),
			}},
			wantContains: [][]string{{
				"**Method:** `BigInstructionData`",
				"- `Data`: 0x00010203",
				"<details><summary>Data</summary>",
				"0x00010203",
				"- `AccountMetaSlice`:",
				"[]",
			}},
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
			wantContains: [][]string{{
				"**Method:** `AccountMut`",
				"- `AccountMetaSlice`:",
				"<details><summary>AccountMetaSlice</summary>",
				"H2qiK1CzW2DheLz9WAGSF1GbvLoqQv9hgS56Rk8Wh3uA",
				"4cubrmdczDbRT8XyBwSR871meZU426S6xkiouzQpspVK",
				"11111111111111111111111111111111",
			}},
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
			wantContains: [][]string{{
				"**Method:** `InitSignatures`",
				"<details><summary>MultisigId</summary>",
				"0x6d636d0000000000000000000000000000000000000000000000000000000000",
				"<details><summary>Root</summary>",
				"0x726f6f7400000000000000000000000000000000000000000000000000000000",
				"- `ValidUntil`: 1767225600",
				"- `TotalSignatures`: 2",
				"- `AccountMetaSlice`:",
				"8UXavXj14P3khJyWSfeDeZ57YS7vo8ynkKemo2M2C1VU",
				"J6fUzHuGEHmqpmmq1BMGfjfeYjPwg4TWsKsJB8WGihoJ",
			}},
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
			wantContains: [][]string{{
				"**Method:** `SetRoot`",
				"- `MultisigId`:",
				"- `Root`:",
				"- `ValidUntil`: 1767225600",
				"- `Metadata`:",
				"<details><summary>Metadata</summary>",
				"chainid: 16423721717087811551",
				"overridepreviousroot: true",
				"<details><summary>MetadataProof</summary>",
				"<details><summary>AccountMetaSlice</summary>",
				"1EMwYGgmo3UPwmyUiPvCUPM5kdL52LHPJXSZNUN1pam",
				"AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd",
				"xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg",
				"FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3",
				"7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd",
				"Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7",
			}},
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
			wantContains: [][]string{{
				"**Method:** `SetConfig`",
				"- `MultisigId`:",
				"- `SignerGroups`:",
				"- `GroupQuorums`:",
				"- `GroupParents`:",
				"- `ClearRoot`: true",
				"<details><summary>AccountMetaSlice</summary>",
				"AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd",
				"xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg",
				"FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3",
				"7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd",
				"Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := describeOperations(tt.ctx, tt.operations)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, len(tt.wantContains), len(got))
				for i, parts := range tt.wantContains {
					for _, p := range parts {
						require.Contains(t, got[i], p)
					}
				}
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

//nolint:paralleltest // call to SetProgramID is not thread-safe
func Test_solanaAnalyzer_describeBatchOperations(t *testing.T) {
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
		name         string
		ctx          ProposalContext
		batchOps     []mcmstypes.BatchOperation
		wantContains [][][]string // batches -> ops -> substrings
		wantErr      string
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
			wantContains: [][][]string{
				{
					{"**Method:** `U8InstructionData`", "- `Data`: 12", "<details><summary>Data</summary>", "12", "- `AccountMetaSlice`:", "[]"},
					{"**Method:** `U8InstructionData`", "- `Data`: 34", "<details><summary>Data</summary>", "34", "- `AccountMetaSlice`:", "[]"},
				},
				{
					{"**Method:** `U8InstructionData`", "- `Data`: 56", "<details><summary>Data</summary>", "56", "- `AccountMetaSlice`:", "[]"},
					{"**Method:** `U8InstructionData`", "- `Data`: 78", "<details><summary>Data</summary>", "78", "- `AccountMetaSlice`:", "[]"},
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
				require.Equal(t, len(tt.wantContains), len(got))
				for bi := range got {
					require.Equal(t, len(tt.wantContains[bi]), len(got[bi]))
					for oi := range got[bi] {
						for _, sub := range tt.wantContains[bi][oi] {
							require.Contains(t, got[bi][oi], sub)
						}
					}
				}
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
