package analyzer

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/ethereum/go-ethereum/common"
	binary "github.com/gagliardetto/binary"
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

var Version1_0_0 = *semver.MustParse("1.0.0")

//nolint:paralleltest // call to SetProgramID is not thread-safe
func TestAnchorInstructionWrapper_TypeID(t *testing.T) {
	t.Parallel()

	// Create a mock instruction that doesn't have BaseVariant field to test error case
	mockInstruction := &mockSolanaInstruction{
		programID: solana.PublicKeyFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}),
		accounts:  []*solana.AccountMeta{},
		data:      []byte{1, 2, 3, 4},
		typeID:    binary.TypeID{1, 2, 3, 4},
		impl:      "test",
	}

	wrapper := &anchorInstructionWrapper{
		anchorInstruction: mockInstruction,
	}

	_, err := wrapper.TypeID()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get BaseVariant field")
}

func TestAnchorInstructionWrapper_ProgramID(t *testing.T) {
	t.Parallel()

	expectedProgramID := solana.PublicKeyFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32})
	mockInstruction := &mockSolanaInstruction{
		programID: expectedProgramID,
		accounts:  []*solana.AccountMeta{},
		data:      []byte{1, 2, 3, 4},
	}

	wrapper := &anchorInstructionWrapper{
		anchorInstruction: mockInstruction,
	}

	programID := wrapper.ProgramID()
	require.Equal(t, expectedProgramID, programID)
}

func TestAnchorInstructionWrapper_Accounts(t *testing.T) {
	t.Parallel()

	expectedAccounts := []*solana.AccountMeta{
		{PublicKey: solana.PublicKeyFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}), IsWritable: true},
		{PublicKey: solana.PublicKeyFromBytes([]byte{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33}), IsWritable: false},
	}

	mockInstruction := &mockSolanaInstruction{
		programID: solana.PublicKeyFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}),
		accounts:  expectedAccounts,
		data:      []byte{1, 2, 3, 4},
	}

	wrapper := &anchorInstructionWrapper{
		anchorInstruction: mockInstruction,
	}

	accounts := wrapper.Accounts()
	require.Equal(t, expectedAccounts, accounts)
}

func TestAnchorInstructionWrapper_Data(t *testing.T) {
	t.Parallel()

	expectedData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	mockInstruction := &mockSolanaInstruction{
		programID: solana.PublicKeyFromBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}),
		accounts:  []*solana.AccountMeta{},
		data:      expectedData,
	}

	wrapper := &anchorInstructionWrapper{
		anchorInstruction: mockInstruction,
	}

	data, err := wrapper.Data()
	require.NoError(t, err)
	require.Equal(t, expectedData, data)
}

func TestYamlDescriptor_MarshalYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "Simple string",
			value: "test value",
			want:  "test value\n",
		},
		{
			name:  "Number",
			value: 42,
			want:  "42\n",
		},
		{
			name:  "Boolean",
			value: true,
			want:  "true\n",
		},
		{
			name:  "Array",
			value: []string{"item1", "item2"},
			want:  "- item1\n- item2\n",
		},
		{
			name:  "Map",
			value: map[string]any{"key": "value"},
			want:  "key: value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			desc := YamlDescriptor{value: tt.value}
			result, err := desc.MarshalYAML()
			require.NoError(t, err)
			require.Equal(t, tt.want, string(result))
		})
	}
}

// Mock implementation for testing
type mockSolanaInstruction struct {
	programID solana.PublicKey
	accounts  []*solana.AccountMeta
	data      []byte
	typeID    binary.TypeID
	impl      any
}

func (m *mockSolanaInstruction) ProgramID() solana.PublicKey {
	return m.programID
}

func (m *mockSolanaInstruction) Accounts() []*solana.AccountMeta {
	return m.accounts
}

func (m *mockSolanaInstruction) Data() ([]byte, error) {
	return m.data, nil
}

func (m *mockSolanaInstruction) Name() string {
	return "MockInstruction"
}

func (m *mockSolanaInstruction) TypeID() (binary.TypeID, error) {
	return m.typeID, nil
}

func (m *mockSolanaInstruction) Impl() (any, error) {
	return m.impl, nil
}

func (m *mockSolanaInstruction) Inputs() []NamedDescriptor {
	return []NamedDescriptor{
		{Name: "test", Value: SimpleDescriptor{Value: "test"}},
	}
}

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
		want       []*DecodedCall // expected DecodedCall results
		wantErr    string
	}{
		{
			name: "success: cpistub.Empty",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewEmptyInstruction()),
			}},
			want: []*DecodedCall{{
				Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
				Method:  "Empty",
				Inputs: []NamedDescriptor{{
					Name:  "AccountMetaSlice",
					Value: YamlDescriptor{value: []string{}},
				}},
			}},
		},
		{
			name: "success: cpistub.U8InstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewU8InstructionDataInstruction(uint8(123))),
			}},
			want: []*DecodedCall{{
				Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
				Method:  "U8InstructionData",
				Inputs: []NamedDescriptor{
					{
						Name:  "Data",
						Value: SimpleDescriptor{Value: "123\n"},
					},
					{
						Name:  "AccountMetaSlice",
						Value: YamlDescriptor{value: []string{}},
					},
				},
			}},
		},
		{
			name: "success: cpistub.StructInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewStructInstructionDataInstruction(cpistub.Value{Value: uint8(45)})),
			}},
			want: []*DecodedCall{{
				Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
				Method:  "StructInstructionData",
				Inputs: []NamedDescriptor{
					{
						Name:  "Data",
						Value: YamlDescriptor{value: cpistub.Value{Value: uint8(45)}},
					},
					{
						Name:  "AccountMetaSlice",
						Value: YamlDescriptor{value: []string{}},
					},
				},
			}},
		},
		{
			name: "success: cpistub.BigInstructionData",
			ctx:  defaultProposalCtx,
			operations: []mcmstypes.Operation{{
				ChainSelector: solanaChainSelector,
				Transaction:   mcmsTxFromInstruction(t, cpistub.NewBigInstructionDataInstruction([]byte{0x0, 0x1, 0x2, 0x3})),
			}},
			want: []*DecodedCall{{
				Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
				Method:  "BigInstructionData",
				Inputs: []NamedDescriptor{
					{
						Name:  "Data",
						Value: YamlDescriptor{value: []byte{0x0, 0x1, 0x2, 0x3}},
					},
					{
						Name:  "AccountMetaSlice",
						Value: YamlDescriptor{value: []string{}},
					},
				},
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
			want: []*DecodedCall{{
				Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
				Method:  "AccountMut",
				Inputs: []NamedDescriptor{{
					Name: "AccountMetaSlice",
					Value: YamlDescriptor{value: []solana.AccountMeta{
						{PublicKey: solana.MPK("H2qiK1CzW2DheLz9WAGSF1GbvLoqQv9hgS56Rk8Wh3uA"), IsWritable: true},
						{PublicKey: solana.MPK("4cubrmdczDbRT8XyBwSR871meZU426S6xkiouzQpspVK"), IsSigner: true},
						{PublicKey: solana.SystemProgramID},
					}},
				}},
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
			want: []*DecodedCall{{
				Address: "Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j",
				Method:  "InitSignatures",
				Inputs: []NamedDescriptor{
					{
						Name:  "MultisigId",
						Value: YamlDescriptor{value: [32]uint8{'m', 'c', 'm'}},
					},
					{
						Name:  "Root",
						Value: YamlDescriptor{value: [32]uint8{'r', 'o', 'o', 't'}},
					},
					{
						Name:  "ValidUntil",
						Value: YamlDescriptor{value: uint32(1767225600)},
					},
					{
						Name:  "TotalSignatures",
						Value: YamlDescriptor{value: uint8(2)},
					},
					{
						Name: "AccountMetaSlice",
						Value: YamlDescriptor{value: []solana.AccountMeta{
							{PublicKey: solana.MPK("8UXavXj14P3khJyWSfeDeZ57YS7vo8ynkKemo2M2C1VU"), IsWritable: true},
							{PublicKey: solana.MPK("J6fUzHuGEHmqpmmq1BMGfjfeYjPwg4TWsKsJB8WGihoJ"), IsWritable: true, IsSigner: true},
							{PublicKey: solana.SystemProgramID},
						}},
					},
				},
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
			want: []*DecodedCall{{
				Address: "Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j",
				Method:  "SetRoot",
				Inputs: []NamedDescriptor{
					{
						Name:  "MultisigId",
						Value: YamlDescriptor{value: [32]uint8{'m', 'c', 'm'}},
					},
					{
						Name:  "Root",
						Value: YamlDescriptor{value: [32]uint8{'r', 'o', 'o', 't'}},
					},
					{
						Name:  "ValidUntil",
						Value: YamlDescriptor{value: uint32(1767225600)},
					},
					{
						Name: "Metadata",
						Value: YamlDescriptor{value: mcm.RootMetadataInput{
							ChainId:              chainsel.SOLANA_DEVNET.Selector,
							Multisig:             solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd"),
							PreOpCount:           1,
							PostOpCount:          2,
							OverridePreviousRoot: true,
						}},
					},
					{
						Name: "MetadataProof",
						Value: YamlDescriptor{value: [][32]uint8{
							common.HexToHash("0x0000000000000000000000000000000000000001"),
							common.HexToHash("0x0000000000000000000000000000000000000002"),
						}},
					},
					{
						Name: "AccountMetaSlice",
						Value: YamlDescriptor{value: []solana.AccountMeta{
							{PublicKey: solana.MPK("1EMwYGgmo3UPwmyUiPvCUPM5kdL52LHPJXSZNUN1pam"), IsWritable: true},
							{PublicKey: solana.MPK("AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd"), IsWritable: true},
							{PublicKey: solana.MPK("xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg"), IsWritable: true},
							{PublicKey: solana.MPK("FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3"), IsWritable: true},
							{PublicKey: solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd")},
							{PublicKey: solana.MPK("Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7"), IsWritable: true, IsSigner: true},
							{PublicKey: solana.SystemProgramID},
						}},
					},
				},
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
			want: []*DecodedCall{{
				Address: "Gp9vJNFpwfRM2M9ebK5pQXEb4ZtWwq66nNRRRRGJwz1j",
				Method:  "SetConfig",
				Inputs: []NamedDescriptor{
					{
						Name:  "MultisigId",
						Value: YamlDescriptor{value: [32]uint8{'m', 'c', 'm'}},
					},
					{
						Name:  "SignerGroups",
						Value: YamlDescriptor{value: []byte{1, 2}},
					},
					{
						Name:  "GroupQuorums",
						Value: YamlDescriptor{value: [32]uint8{3, 4, 5}},
					},
					{
						Name:  "GroupParents",
						Value: YamlDescriptor{value: [32]uint8{6, 7, 8}},
					},
					{
						Name:  "ClearRoot",
						Value: YamlDescriptor{value: true},
					},
					{
						Name: "AccountMetaSlice",
						Value: YamlDescriptor{value: []solana.AccountMeta{
							{PublicKey: solana.MPK("AE4UPuh9q1ZCzqzqicw1YujuLC35oTpi1JCpcK6KojPd"), IsWritable: true},
							{PublicKey: solana.MPK("xZzLbR8t1jbHia2nQoRUyhKL7WvjDXRdUQqwzbEVTvg"), IsWritable: true},
							{PublicKey: solana.MPK("FjkJnFj82vM8zq2SEes1WV4ZFEkruPZCcpkXpL92Qhy3"), IsWritable: true},
							{PublicKey: solana.MPK("7eJ2ZKsx3ie1vR1bFaGp4pB5iatjUAfDPtgFDE2sXkZd"), IsWritable: true},
							{PublicKey: solana.MPK("Frr7euo9xRokH9pSmpFf2YbHWB4W3w2Jh7r7hZiu4PD7"), IsWritable: true, IsSigner: true},
							{PublicKey: solana.SystemProgramID},
						}},
					},
				},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test the core analysis function directly instead of the high-level report function
			results, err := AnalyzeSolanaTransactions(tt.ctx, uint64(solanaChainSelector), []mcmstypes.Transaction{tt.operations[0].Transaction})

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Len(t, results, len(tt.want), "Number of results should match expected")

				// Compare each DecodedCall
				for i, result := range results {
					expected := tt.want[i]
					require.Equal(t, expected.Address, result.Address, "Address mismatch for result %d", i)
					require.Equal(t, expected.Method, result.Method, "Method mismatch for result %d", i)
					require.Len(t, result.Inputs, len(expected.Inputs), "Number of inputs should match for result %d", i)

					// Compare each input
					for j, input := range result.Inputs {
						expectedInput := expected.Inputs[j]
						require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for result %d, input %d", i, j)
						require.Equal(t, expectedInput.Value.Describe(nil), input.Value.Describe(nil), "Input value mismatch for result %d, input %d", i, j)
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
		name     string
		ctx      ProposalContext
		batchOps []mcmstypes.BatchOperation
		want     []*DecodedCall // expected DecodedCall results
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
			want: []*DecodedCall{
				{
					Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
					Method:  "U8InstructionData",
					Inputs: []NamedDescriptor{
						{
							Name:  "Data",
							Value: SimpleDescriptor{Value: "12\n"},
						},
						{
							Name:  "AccountMetaSlice",
							Value: YamlDescriptor{value: []string{}},
						},
					},
				},
				{
					Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
					Method:  "U8InstructionData",
					Inputs: []NamedDescriptor{
						{
							Name:  "Data",
							Value: SimpleDescriptor{Value: "34\n"},
						},
						{
							Name:  "AccountMetaSlice",
							Value: YamlDescriptor{value: []string{}},
						},
					},
				},
				{
					Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
					Method:  "U8InstructionData",
					Inputs: []NamedDescriptor{
						{
							Name:  "Data",
							Value: SimpleDescriptor{Value: "56\n"},
						},
						{
							Name:  "AccountMetaSlice",
							Value: YamlDescriptor{value: []string{}},
						},
					},
				},
				{
					Address: "2zZwzyptLqwFJFEFxjPvrdhiGpH9pJ3MfrrmZX6NTKxm",
					Method:  "U8InstructionData",
					Inputs: []NamedDescriptor{
						{
							Name:  "Data",
							Value: SimpleDescriptor{Value: "78\n"},
						},
						{
							Name:  "AccountMetaSlice",
							Value: YamlDescriptor{value: []string{}},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test the core analysis function directly instead of the high-level report function
			var allResults []*DecodedCall
			var err error

			// Analyze all transactions from all batches
			for _, batch := range tt.batchOps {
				results, batchErr := AnalyzeSolanaTransactions(tt.ctx, uint64(batch.ChainSelector), batch.Transactions)
				if batchErr != nil {
					err = batchErr
					break
				}
				allResults = append(allResults, results...)
			}

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Len(t, allResults, len(tt.want), "Number of results should match expected")

				// Compare each DecodedCall
				for i, result := range allResults {
					expected := tt.want[i]
					require.Equal(t, expected.Address, result.Address, "Address mismatch for result %d", i)
					require.Equal(t, expected.Method, result.Method, "Method mismatch for result %d", i)
					require.Len(t, result.Inputs, len(expected.Inputs), "Number of inputs should match for result %d", i)

					// Compare each input
					for j, input := range result.Inputs {
						expectedInput := expected.Inputs[j]
						require.Equal(t, expectedInput.Name, input.Name, "Input name mismatch for result %d, input %d", i, j)
						require.Equal(t, expectedInput.Value.Describe(nil), input.Value.Describe(nil), "Input value mismatch for result %d, input %d", i, j)
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
