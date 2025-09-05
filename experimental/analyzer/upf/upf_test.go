package upf

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/go-cmp/cmp"
	mcmsbindings "github.com/smartcontractkit/ccip-owner-contracts/pkg/gethwrappers"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_6_0/rmn_remote"
	rmnremotebindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/rmn_remote"
	timelockbindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/timelock"
	"github.com/stretchr/testify/require"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	cldfds "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	mcmsanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

func TestUpfConvertTimelockProposal(t *testing.T) {
	t.Parallel()
	ds := cldfds.NewMemoryDataStore()

	mustAdd := func(chain uint64, addr, typeAndVersion string) {
		tv := cldf.MustTypeAndVersionFromString(typeAndVersion)
		storeAddr := addr
		if strings.HasPrefix(addr, "0x") {
			storeAddr = addr
		}
		ref := cldfds.AddressRef{
			ChainSelector: chain,
			Address:       storeAddr,
			Type:          cldfds.ContractType(tv.Type),
			Version:       &tv.Version,
			// Use address+type as a unique Qualifier (avoids clashes)
			Qualifier: fmt.Sprintf("%s-%s", addr, tv.Type),
		}
		if !tv.Labels.IsEmpty() {
			ref.Labels = cldfds.NewLabelSet(tv.Labels.List()...)
		}
		require.NoError(t, ds.Addresses().Add(ref))
	}

	// ---- EVM: ethereum-testnet-sepolia-base-1
	mustAdd(10344971235874465080, "0x5f077BCeE6e285154473F65699d6F46Fd03D105A", "RBACTimelock 1.0.0")
	mustAdd(10344971235874465080, "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953", "ProposerManyChainMultisig 1.0.0")
	mustAdd(10344971235874465080, "0x76B12C4f3672aA613F1b2302327827B7B74064E1", "RMNRemote 1.6.0")

	// ---- EVM: bsc-testnet
	mustAdd(13264668187771770619, "0x804759c9bdd258A810987FDe21c9E24C5383b722", "RBACTimelock 1.0.0")
	mustAdd(13264668187771770619, "0x9A60462e4CA802E3E945663930Be0d162e662091", "ProposerManyChainMultisig 1.0.0")
	mustAdd(13264668187771770619, "0x76B12C4f3672aA613F1b2302327827B7B74064E1", "RMNRemote 1.6.0")

	// ---- Solana: devnet
	mustAdd(16423721717087811551, "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk", "ManyChainMultiSigProgram 1.0.0")
	mustAdd(16423721717087811551, "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0", "ProposerManyChainMultiSig 1.0.0")
	mustAdd(16423721717087811551, "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA", "RBACTimelockProgram 1.0.0")
	mustAdd(16423721717087811551, "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV", "RBACTimelock 1.0.0")
	mustAdd(16423721717087811551, "FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq", "ProposerAccessControllerAccount 1.0.0")
	mustAdd(16423721717087811551, "2hABoxD9U5A4j4x3kNDf4dBJ7ZP384Zbs3TuFn9QUTSs", "CancellerAccessControllerAccount 1.0.0")
	mustAdd(16423721717087811551, "68ds9kDfB6rJfC4zzeeQ8E9ZMwqSzFQEie1886VAPn68", "BypasserAccessControllerAccount 1.0.0")
	mustAdd(16423721717087811551, "RmnXLft1mSEwDgMKu2okYuHkiazxntFFcZFrrcXxYg7", "RMNRemote 1.0.0")

	env := cldf.Environment{
		DataStore:         ds.Seal(),
		ExistingAddresses: cldf.NewMemoryAddressBook(),
	}

	proposalCtx, err := mcmsanalyzer.NewDefaultProposalContext(
		env,
		mcmsanalyzer.WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0":              mcmsbindings.RBACTimelockABI,
			"RMNRemote 1.6.0":                 rmn_remote.RMNRemoteABI,
			"ProposerManyChainMultisig 1.0.0": mcmsbindings.ManyChainMultiSigABI,
		}),
		mcmsanalyzer.WithSolanaDecoders(map[string]mcmsanalyzer.DecodeInstructionFn{
			"RBACTimelockProgram 1.0.0": mcmsanalyzer.DIFn(timelockbindings.DecodeInstruction),
			"RMNRemote 1.0.0":           mcmsanalyzer.DIFn(rmnremotebindings.DecodeInstruction),
		}),
	)
	require.NoError(t, err)

	tests := []struct {
		name             string
		timelockProposal string
		signers          map[mcmstypes.ChainSelector][]common.Address
		want             string
		wantErr          string
	}{
		{
			name:             "simple proposal - RMN curse",
			timelockProposal: timelockProposalRMNCurse,
			want:             upfProposalRMNCurse,
			signers: map[mcmstypes.ChainSelector][]common.Address{
				mcmstypes.ChainSelector(chainsel.ETHEREUM_TESTNET_SEPOLIA_BASE_1.Selector): {
					common.HexToAddress("0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"),
					common.HexToAddress("0x9A60462e4CA802E3E945663930Be0d162e662091"),
					common.HexToAddress("0x5f077BCeE6e285154473F65699d6F46Fd03D105A"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timelockProposal, err := mcms.NewTimelockProposal(strings.NewReader(tt.timelockProposal))
			require.NoError(t, err)
			mcmProposal := convertTimelockProposal(t.Context(), t, timelockProposal)

			got, err := UpfConvertTimelockProposal(proposalCtx, timelockProposal, mcmProposal, tt.signers)
			// err2 := os.WriteFile("/tmp/got.yaml", []byte(got), 0600)
			// require.NoError(t, err2)
			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Empty(t, cmp.Diff(tt.want, got))
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

// ----- helpers -----

func convertTimelockProposal(ctx context.Context, t *testing.T, timelockProposal *mcms.TimelockProposal) *mcms.Proposal {
	t.Helper()

	converters := make(map[mcmstypes.ChainSelector]mcmssdk.TimelockConverter)
	for chain := range timelockProposal.ChainMetadata {
		chainFamily, err := mcmstypes.GetChainSelectorFamily(chain)
		require.NoError(t, err)

		switch chainFamily {
		case chainsel.FamilyEVM:
			converters[chain] = &mcmsevmsdk.TimelockConverter{}
		case chainsel.FamilySolana:
			converters[chain] = mcmssolanasdk.TimelockConverter{}
		default:
			t.Fatalf("unsupported chain family %s", chainFamily)
		}
	}

	mcmProposal, _, err := timelockProposal.Convert(ctx, converters)
	require.NoError(t, err)

	return &mcmProposal
}

// ----- data -----
var timelockProposalRMNCurse = `{
  "version": "v1",
  "kind": "TimelockProposal",
  "validUntil": 1999999999,
  "signatures": [],
  "overridePreviousRoot": false,
  "chainMetadata": {
    "10344971235874465080": {
      "startingOpCount": 1,
      "mcmAddress": "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953",
      "additionalFields": null
    },
    "13264668187771770619": {
      "startingOpCount": 2,
      "mcmAddress": "0x9A60462e4CA802E3E945663930Be0d162e662091",
      "additionalFields": null
    },
    "16423721717087811551": {
      "startingOpCount": 3,
      "mcmAddress": "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0",
      "additionalFields": {
        "proposerRoleAccessController": "FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq",
        "cancellerRoleAccessController": "2hABoxD9U5A4j4x3kNDf4dBJ7ZP384Zbs3TuFn9QUTSs",
        "bypasserRoleAccessController": "68ds9kDfB6rJfC4zzeeQ8E9ZMwqSzFQEie1886VAPn68"
      }
    }
  },
  "description": "simple EVM proposal with RMN curse",
  "action": "schedule",
  "delay": "5m0s",
  "timelockAddresses": {
    "10344971235874465080": "0x5f077BCeE6e285154473F65699d6F46Fd03D105A",
    "13264668187771770619": "0x804759c9bdd258A810987FDe21c9E24C5383b722",
    "16423721717087811551": "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV"
  },
  "operations": [
    {
      "chainSelector": 10344971235874465080,
      "transactions": [
        {
          "contractType": "RMNRemote",
          "tags": [],
          "to": "0x76B12C4f3672aA613F1b2302327827B7B74064E1",
          "data": "+LuHbgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAALgVkXADj5b7AAAAAAAAAAAAAAAAAAAAAA==",
          "additionalFields": { "value": 0 }
        }
      ]
    },
    {
      "chainSelector": 13264668187771770619,
      "transactions": [
        {
          "contractType": "",
          "tags": [],
          "to": "0x76B12C4f3672aA613F1b2302327827B7B74064E1",
          "data": "+LuHbgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAI+QuIdt7mU4AAAAAAAAAAAAAAAAAAAAAA==",
          "additionalFields": { "value": 0 }
        }
      ]
    },
    {
      "chainSelector": 16423721717087811551,
      "transactions": [
        {
          "contractType": "RMNRemote",
          "tags": [],
          "to": "RmnXLft1mSEwDgMKu2okYuHkiazxntFFcZFrrcXxYg7",
          "data": "Cn/i44oDwEn7lo8DcJEVuAAAAAAAAAAA",
          "additionalFields": {
            "accounts": [
              {
                "PublicKey": "GbQFCDTPbhwPjeUvhu7hXEM3Wm2S6t3FCxoCmQtKxetw",
                "IsWritable": false,
                "IsSigner": false
              },
              {
                "PublicKey": "35u11sTYbcen34onkPHVEaekkJ4ua4k1SqkXV2x7bEPy",
                "IsWritable": true,
                "IsSigner": false
              },
              {
                "PublicKey": "CpbeEvmTR4UE8CgDDL5b1nqjSz7JCD4wNJhxPLZRkSL1",
                "IsWritable": true,
                "IsSigner": false
              },
              {
                "PublicKey": "11111111111111111111111111111111",
                "IsWritable": false,
                "IsSigner": false
              }
            ],
            "value": 0
          }
        }
      ]
    }
  ]
}`

var upfProposalRMNCurse = `---
msigType: mcms
proposalHash: "0x41ce69645a9ce865c1035dc310e49bcb8057e932d20f50879858fc0b3319f909"
mcmsParams:
  validUntil: 1999999999
  merkleRoot: "0x963d51589fcf57be4be3f35dd42a9519deaf47754b81a61b2ef48475fb1824bf"
  asciiProposalHash: '\xfc&\x9b\xef; \xc6R\xde\xba\x97\xe8\xcd!\x9e\xb3\xe4ya\x99\x17\x8b\x10\xefqY\x82\xe4]\x7f\xd3\xd3'
  overridePreviousRoot: false
transactions:
- index: 0
  chainFamily: evm
  chainId: "84532"
  chainName: ethereum-testnet-sepolia-base-1
  chainShortName: ethereum-testnet-sepolia-base-1
  msigAddress: "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"
  timelockAddress: "0x5f077BCeE6e285154473F65699d6F46Fd03D105A"
  to: "0x5f077BCeE6e285154473F65699d6F46Fd03D105A"
  value: 0
  data: "0xa944142d00000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000773593ff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000012c0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000076b12c4f3672aa613f1b2302327827b7b74064e1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000064f8bb876e000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000010000000000000000b8159170038f96fb0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
  txNonce: 2
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: function scheduleBatch((address,uint256,bytes)[] calls, bytes32 predecessor, bytes32 salt, uint256 delay) returns()
      functionArgs:
        calls:
        - to: "0x76B12C4f3672aA613F1b2302327827B7B74064E1"
          value: 0
          data:
            functionName: function curse(bytes16[] subjects) returns()
            functionArgs:
              subjects:
              - "0x0000000000000000b8159170038f96fb"
        delay: "300"
        predecessor: "0x0000000000000000000000000000000000000000000000000000000000000000"
        salt: "0x773593ff00000000000000000000000000000000000000000000000000000000"
- index: 1
  chainFamily: evm
  chainId: "97"
  chainName: binance_smart_chain-testnet
  chainShortName: binance_smart_chain-testnet
  msigAddress: "0x9A60462e4CA802E3E945663930Be0d162e662091"
  timelockAddress: "0x804759c9bdd258A810987FDe21c9E24C5383b722"
  to: "0x804759c9bdd258A810987FDe21c9E24C5383b722"
  value: 0
  data: "0xa944142d00000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000000773593ff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000012c0000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000076b12c4f3672aa613f1b2302327827b7b74064e1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000600000000000000000000000000000000000000000000000000000000000000064f8bb876e0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000100000000000000008f90b8876dee65380000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
  txNonce: 3
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: function scheduleBatch((address,uint256,bytes)[] calls, bytes32 predecessor, bytes32 salt, uint256 delay) returns()
      functionArgs:
        calls:
        - to: "0x76B12C4f3672aA613F1b2302327827B7B74064E1"
          value: 0
          data:
            functionName: function curse(bytes16[] subjects) returns()
            functionArgs:
              subjects:
              - "0x00000000000000008f90b8876dee6538"
        delay: "300"
        predecessor: "0x0000000000000000000000000000000000000000000000000000000000000000"
        salt: "0x773593ff00000000000000000000000000000000000000000000000000000000"
- index: 2
  chainFamily: solana
  chainId: EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG
  chainName: solana-devnet
  chainShortName: solana-devnet
  msigAddress: 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0
  timelockAddress: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV
  to: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA
  value: 0
  data: D2DZq3wEcfNFNFI2TndnMUs4WnZpNk1jTGRrYUdERDVDbFgxS2t5VpAXlZ2LYPhZ+p8F9JucBPQaESwj/lQ3CwCjnNzLdfsEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB3NZP/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAA=
  txNonce: 4
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: InitializeOperation
      functionArgs:
        AccountMetaSlice:
        - 25G4MQqNZdptySDgzE3bQtNSbYMSjzF24V6pVVSdaj2S   [writable]
        - DJnQbX4PrA4854CpsA5hDkizx1drAccQbHE8jZrgRhAk
        - FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq
        - 9K5QmiFUayo3Hsmjt47VgJ8HeWuVgrUToAk9Huw5XmLk   [writable]
        - 11111111111111111111111111111111
        Id: 0x9017959d8b60f859fa9f05f49b9c04f41a112c23fe54370b00a39cdccb75fb04
        InstructionCount: 1
        Predecessor: 0x0000000000000000000000000000000000000000000000000000000000000000
        Salt: 0x773593ff00000000000000000000000000000000000000000000000000000000
        TimelockId: 0x453452364e7767314b385a7669364d634c646b6147444435436c58314b6b7956
- index: 3
  chainFamily: solana
  chainId: EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG
  chainName: solana-devnet
  chainShortName: solana-devnet
  msigAddress: 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0
  timelockAddress: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV
  to: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA
  value: 0
  data: w+bVh5CUjlVFNFI2TndnMUs4WnZpNk1jTGRrYUdERDVDbFgxS2t5VpAXlZ2LYPhZ+p8F9JucBPQaESwj/lQ3CwCjnNzLdfsEBliT7ZWrhugwWyiYp2G+HCHNv7C39ebv1T6DsoMrGlAEAAAA569Vk65pFF7HVjX5akNsLQQfz9+AaloAzqNrzzoGc1AAAB74fnS+SLFE6OvgAxbo+S+KqA2nm/gNrv6H0f98BW1YAAGvogYtczqE0C5vP92khgtsL3GSUtW9S5XWTvA81tlPygABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==
  txNonce: 5
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: InitializeInstruction
      functionArgs:
        AccountMetaSlice: []
        Accounts:
        - pubkey: GbQFCDTPbhwPjeUvhu7hXEM3Wm2S6t3FCxoCmQtKxetw
          issigner: false
          iswritable: false
        - pubkey: 35u11sTYbcen34onkPHVEaekkJ4ua4k1SqkXV2x7bEPy
          issigner: false
          iswritable: true
        - pubkey: CpbeEvmTR4UE8CgDDL5b1nqjSz7JCD4wNJhxPLZRkSL1
          issigner: false
          iswritable: true
        - pubkey: 11111111111111111111111111111111
          issigner: false
          iswritable: false
        Id: 0x9017959d8b60f859fa9f05f49b9c04f41a112c23fe54370b00a39cdccb75fb04
        ProgramId: RmnXLft1mSEwDgMKu2okYuHkiazxntFFcZFrrcXxYg7
        TimelockId: 0x453452364e7767314b385a7669364d634c646b6147444435436c58314b6b7956
- index: 4
  chainFamily: solana
  chainId: EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG
  chainName: solana-devnet
  chainShortName: solana-devnet
  msigAddress: 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0
  timelockAddress: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV
  to: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA
  value: 0
  data: TE1mg4gMLQVFNFI2TndnMUs4WnZpNk1jTGRrYUdERDVDbFgxS2t5VpAXlZ2LYPhZ+p8F9JucBPQaESwj/lQ3CwCjnNzLdfsEAAAAABgAAAAKf+LjigPASfuWjwNwkRW4AAAAAAAAAAA=
  txNonce: 6
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: AppendInstructionData
      functionArgs:
        AccountMetaSlice:
        - 25G4MQqNZdptySDgzE3bQtNSbYMSjzF24V6pVVSdaj2S   [writable]
        - DJnQbX4PrA4854CpsA5hDkizx1drAccQbHE8jZrgRhAk
        - FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq
        - 9K5QmiFUayo3Hsmjt47VgJ8HeWuVgrUToAk9Huw5XmLk   [writable]
        - 11111111111111111111111111111111
        Id: 0x9017959d8b60f859fa9f05f49b9c04f41a112c23fe54370b00a39cdccb75fb04
        IxDataChunk: 0x0a7fe2e38a03c049fb968f03709115b80000000000000000
        IxIndex: 0
        TimelockId: 0x453452364e7767314b385a7669364d634c646b6147444435436c58314b6b7956
- index: 5
  chainFamily: solana
  chainId: EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG
  chainName: solana-devnet
  chainShortName: solana-devnet
  msigAddress: 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0
  timelockAddress: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV
  to: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA
  value: 0
  data: P9AgYlW27IxFNFI2TndnMUs4WnZpNk1jTGRrYUdERDVDbFgxS2t5VpAXlZ2LYPhZ+p8F9JucBPQaESwj/lQ3CwCjnNzLdfsE
  txNonce: 7
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: FinalizeOperation
      functionArgs:
        AccountMetaSlice:
        - 25G4MQqNZdptySDgzE3bQtNSbYMSjzF24V6pVVSdaj2S   [writable]
        - DJnQbX4PrA4854CpsA5hDkizx1drAccQbHE8jZrgRhAk
        - FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq
        - 9K5QmiFUayo3Hsmjt47VgJ8HeWuVgrUToAk9Huw5XmLk   [writable]
        Id: 0x9017959d8b60f859fa9f05f49b9c04f41a112c23fe54370b00a39cdccb75fb04
        TimelockId: 0x453452364e7767314b385a7669364d634c646b6147444435436c58314b6b7956
- index: 6
  chainFamily: solana
  chainId: EtWTRABZaYq6iMfeYKouRu166VU2xqa1wcaWoxPkrZBG
  chainName: solana-devnet
  chainShortName: solana-devnet
  msigAddress: 5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0
  timelockAddress: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV
  to: DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA
  value: 0
  data: 8oxXakfiViBFNFI2TndnMUs4WnZpNk1jTGRrYUdERDVDbFgxS2t5VpAXlZ2LYPhZ+p8F9JucBPQaESwj/lQ3CwCjnNzLdfsELAEAAAAAAAA=
  txNonce: 8
  metadata:
    contractType: RBACTimelock
    decodedCalldata:
      functionName: ScheduleBatch
      functionArgs:
        AccountMetaSlice:
        - 25G4MQqNZdptySDgzE3bQtNSbYMSjzF24V6pVVSdaj2S   [writable]
        - DJnQbX4PrA4854CpsA5hDkizx1drAccQbHE8jZrgRhAk
        - FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq
        - 9K5QmiFUayo3Hsmjt47VgJ8HeWuVgrUToAk9Huw5XmLk   [writable]
        Delay: 300
        Id: 0x9017959d8b60f859fa9f05f49b9c04f41a112c23fe54370b00a39cdccb75fb04
        TimelockId: 0x453452364e7767314b385a7669364d634c646b6147444435436c58314b6b7956
        calls:
        - to: RmnXLft1mSEwDgMKu2okYuHkiazxntFFcZFrrcXxYg7
          value: 0
          data:
            functionName: Curse
            functionArgs:
              AccountMetaSlice:
              - GbQFCDTPbhwPjeUvhu7hXEM3Wm2S6t3FCxoCmQtKxetw
              - 35u11sTYbcen34onkPHVEaekkJ4ua4k1SqkXV2x7bEPy   [writable]
              - CpbeEvmTR4UE8CgDDL5b1nqjSz7JCD4wNJhxPLZRkSL1   [writable]
              - 11111111111111111111111111111111
              Subject:
                value: 0xfb968f03709115b80000000000000000
signers:
  "10344971235874465080":
  - "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"
  - "0x9A60462e4CA802E3E945663930Be0d162e662091"
  - "0x5f077BCeE6e285154473F65699d6F46Fd03D105A"
`
