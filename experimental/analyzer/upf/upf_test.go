package upf

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/go-cmp/cmp"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_6_0/rmn_remote"
	rmnremotebindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/rmn_remote"
	timelockbindings "github.com/smartcontractkit/chainlink-ccip/chains/solana/gobindings/v0_1_0/timelock"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/lib/access/rbac"
	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"

	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmssdk "github.com/smartcontractkit/mcms/sdk"
	mcmsevmsdk "github.com/smartcontractkit/mcms/sdk/evm"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
	mcmstonsdk "github.com/smartcontractkit/mcms/sdk/ton"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"

	mcmsanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

func TestUpfConvertTimelockProposal(t *testing.T) {
	t.Parallel()
	ds := datastore.NewMemoryDataStore()

	// ---- EVM: ethereum-testnet-sepolia-base-1
	dsAddContract(t, ds, 10344971235874465080, "0x5f077BCeE6e285154473F65699d6F46Fd03D105A", "RBACTimelock 1.0.0")
	dsAddContract(t, ds, 10344971235874465080, "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953", "ProposerManyChainMultisig 1.0.0")
	dsAddContract(t, ds, 10344971235874465080, "0x76B12C4f3672aA613F1b2302327827B7B74064E1", "RMNRemote 1.6.0")

	// ---- EVM: bsc-testnet
	dsAddContract(t, ds, 13264668187771770619, "0x804759c9bdd258A810987FDe21c9E24C5383b722", "RBACTimelock 1.0.0")
	dsAddContract(t, ds, 13264668187771770619, "0x9A60462e4CA802E3E945663930Be0d162e662091", "ProposerManyChainMultisig 1.0.0")
	dsAddContract(t, ds, 13264668187771770619, "0x76B12C4f3672aA613F1b2302327827B7B74064E1", "RMNRemote 1.6.0")

	// ---- Solana: devnet
	dsAddContract(t, ds, 16423721717087811551, "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk", "ManyChainMultiSigProgram 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "5vNJx78mz7KVMjhuipyr9jKBKcMrKYGdjGkgE4LUmjKk.ID6xwqkfFkH6dx2LF0O2NKfHKwHywEB0", "ProposerManyChainMultiSig 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA", "RBACTimelockProgram 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "DoajfR5tK24xVw51fWcawUZWhAXD8yrBJVacc13neVQA.E4R6Nwg1K8Zvi6McLdkaGDD5ClX1KkyV", "RBACTimelock 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "FTDusxFg9NmmFGRg5jfA9nHCiCpZ7dJktawfRBcUBhq", "ProposerAccessControllerAccount 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "2hABoxD9U5A4j4x3kNDf4dBJ7ZP384Zbs3TuFn9QUTSs", "CancellerAccessControllerAccount 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "68ds9kDfB6rJfC4zzeeQ8E9ZMwqSzFQEie1886VAPn68", "BypasserAccessControllerAccount 1.0.0")
	dsAddContract(t, ds, 16423721717087811551, "RmnXLft1mSEwDgMKu2okYuHkiazxntFFcZFrrcXxYg7", "RMNRemote 1.0.0")

	env := deployment.Environment{
		DataStore:         ds.Seal(),
		ExistingAddresses: deployment.NewMemoryAddressBook(),
	}

	proposalCtx, err := mcmsanalyzer.NewDefaultProposalContext(
		env,
		mcmsanalyzer.WithEVMABIMappings(map[string]string{
			"RBACTimelock 1.0.0":              mcmsanalyzer.RBACTimelockMetaDataTesting.ABI,
			"RMNRemote 1.6.0":                 rmn_remote.RMNRemoteABI,
			"ProposerManyChainMultisig 1.0.0": mcmsanalyzer.ManyChainMultiSigMetaData.ABI,
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

			got, err := UpfConvertTimelockProposal(t.Context(), proposalCtx, env, timelockProposal, mcmProposal, tt.signers)
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
		case chainsel.FamilySui:
			converter, err := mcmssuisdk.NewTimelockConverter()
			require.NoError(t, err)
			converters[chain] = converter
		case chainsel.FamilyTon:
			converters[chain] = mcmstonsdk.NewTimelockConverter()
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
  txNonce: 1
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
  txNonce: 3
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
  txNonce: 4
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
  txNonce: 5
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
  txNonce: 6
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
  txNonce: 7
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

var timelockProposalSui = `{
  "version": "v1",
  "kind": "TimelockProposal",
  "validUntil": 1999999999,
  "signatures": [],
  "overridePreviousRoot": false,
  "chainMetadata": {
    "9762610643973837292": {
      "startingOpCount": 1,
      "mcmAddress": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d",
      "additionalFields": null
    }
  },
  "description": "simple Sui proposal",
  "action": "schedule",
  "delay": "5m0s",
  "timelockAddresses": {
    "9762610643973837292": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  },
  "operations": [
    {
      "chainSelector": 9762610643973837292,
      "transactions": [
        {
          "contractType": "MCMSUser",
          "tags": [],
          "to": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d",
          "data": "i8WcKEL0NsEiFpGjWdxClBwfJeyhP0ut55f7Dg2PS2W5dbWeXl19LUYyEaeuZRHbtJS9IbqY1GNZHOkUhofVcGRhdGVkIEZpZWxkIEEKAQIDBAUGBwgJCg==",
          "additionalFields": {
            "module_name": "mcms_user",
            "function": "function_one",
            "state_obj": "0x8bc59c2842f436c1221691a359dc42941c1f25eca13f4bad79f7b00e8df4b968"
          }
        }
      ]
    }
  ]
}`

var upfProposalSui = `---
msigType: mcms
proposalHash: "0x1c733d9d09e9d41e1651596078df88b00c68e085cc6bf14b8f346866b1741a28"
mcmsParams:
  validUntil: 1999999999
  merkleRoot: "0xeeaa854482fdd28dec1ca358c4ba9c7399560b580683c7fa372e9a69eab8ba1d"
  asciiProposalHash: '\x93>\x07\xb8>\xce3\xfa\xa7\xccZ\x1e\xea\xf8|\xb39\x9c\x10s\xd7\x98\xc8\xa6\x1d\xe13\x99\xa1u\xe2.'
  overridePreviousRoot: false
transactions:
- index: 0
  chainFamily: sui
  chainId: "2"
  chainName: sui-testnet
  chainShortName: sui-testnet
  msigAddress: "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  timelockAddress: "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  to: ""
  value: 0
  data: AU6CWkdYBk33E3YuQxw6FrgQWFcZUhRGnbDWmFt9cCZtAQltY21zX3VzZXIBDGZ1bmN0aW9uX29uZQFYi8WcKEL0NsEiFpGjWdxClBwfJeyhP0ut55f7Dg2PS2W5dbWeXl19LUYyEaeuZRHbtJS9IbqY1GNZHOkUhofVcGRhdGVkIEZpZWxkIEEKAQIDBAUGBwgJCiAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACB3NZP/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACwBAAAAAAAA
  txNonce: 1
  metadata:
    contractType: MCMS
    decodedCalldata:
      functionName: "failed to decode Sui transaction: could not find function in contractInterfaces for mcms::timelock_schedule_batch"
      functionArgs: {}
signers:
  "9762610643973837292":
  - "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"
`

var timelockProposalSuiUnknownModule = `{
  "version": "v1",
  "kind": "TimelockProposal",
  "validUntil": 1999999999,
  "signatures": [],
  "overridePreviousRoot": false,
  "chainMetadata": {
    "9762610643973837292": {
      "startingOpCount": 1,
      "mcmAddress": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d",
      "additionalFields": null
    }
  },
  "description": "Sui proposal with unknown module",
  "action": "schedule",
  "delay": "5m0s",
  "timelockAddresses": {
    "9762610643973837292": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  },
  "operations": [
    {
      "chainSelector": 9762610643973837292,
      "transactions": [
        {
          "contractType": "MCMSUser",
          "tags": [],
          "to": "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d",
          "data": "c29tZSBkYXRh",
          "additionalFields": {
            "module_name": "unknown_module",
            "function": "some_function",
            "state_obj": "0x123"
          }
        }
      ]
    }
  ]
}`

var upfProposalSuiUnknownModule = `---
msigType: mcms
proposalHash: "0x5433c70ce0b94602235ae03d5485a3ff991b90d35b90f3474af5455f1105c198"
mcmsParams:
  validUntil: 1999999999
  merkleRoot: "0x0104cddb47805604d82eeab0e02cb33c4374c1e635ab038d2a1ed9038c48e4a9"
  asciiProposalHash: 'L\xb9E\x9d\xfeMY\x83\xec3\xba\x00\xa6F0@\x82 \xd4\xc0\x9bj-"C\xcb\xf6\xb6v\xc0B\xbc'
  overridePreviousRoot: false
transactions:
- index: 0
  chainFamily: sui
  chainId: "2"
  chainName: sui-testnet
  chainShortName: sui-testnet
  msigAddress: "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  timelockAddress: "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d"
  to: ""
  value: 0
  data: AU6CWkdYBk33E3YuQxw6FrgQWFcZUhRGnbDWmFt9cCZtAQ51bmtub3duX21vZHVsZQENc29tZV9mdW5jdGlvbgEJc29tZSBkYXRhIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIHc1k/8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAALAEAAAAAAAA=
  txNonce: 1
  metadata:
    contractType: MCMS
    decodedCalldata:
      functionName: "failed to decode Sui transaction: could not find function in contractInterfaces for mcms::timelock_schedule_batch"
      functionArgs: {}
signers:
  "9762610643973837292":
  - "0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"
`

// timelockProposalTon is generated using makeTONGrantRoleTx helper
var timelockProposalTon = func() string {
	// Create a GrantRole transaction for the test
	targetAddr := address.MustParseAddr("EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8")
	exampleRole := crypto.Keccak256Hash([]byte("EXAMPLE_ROLE"))
	exampleRoleBig, _ := cell.BeginCell().
		MustStoreBigInt(new(big.Int).SetBytes(exampleRole[:]), 257).
		EndCell().
		ToBuilder().
		ToSlice().
		LoadBigInt(256)

	grantRoleData, _ := tlb.ToCell(rbac.GrantRole{
		QueryID: 1,
		Role:    exampleRoleBig,
		Account: targetAddr,
	})

	tx, _ := mcmstonsdk.NewTransaction(
		targetAddr,
		grantRoleData.ToBuilder().ToSlice(),
		big.NewInt(0),
		"com.chainlink.ton.lib.access.RBAC",
		[]string{"grantRole"},
	)

	// Marshal the transaction data
	txData, _ := json.Marshal(tx)
	var txMap map[string]interface{}
	_ = json.Unmarshal(txData, &txMap)

	return fmt.Sprintf(`{
  "version": "v1",
  "kind": "TimelockProposal",
  "validUntil": 1999999999,
  "signatures": [],
  "overridePreviousRoot": false,
  "chainMetadata": {
    "1399300952838017768": {
      "startingOpCount": 1,
      "mcmAddress": "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8",
      "additionalFields": null
    }
  },
  "description": "simple TON proposal with GrantRole",
  "action": "schedule",
  "delay": "5m0s",
  "timelockAddresses": {
    "1399300952838017768": "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8"
  },
  "operations": [
    {
      "chainSelector": 1399300952838017768,
      "transactions": [%s]
    }
  ]
}`, string(txData))
}()

func TestUpfConvertTimelockProposalWithSui(t *testing.T) {
	t.Parallel()
	ds := datastore.NewMemoryDataStore()

	// ---- Sui: testnet
	dsAddContract(t, ds, chainsel.SUI_TESTNET.Selector, "0x4e825a4758064df713762e431c3a16b8105857195214469db0d6985b7d70266d", "MCMSUser 1.0.0")

	env := deployment.Environment{
		DataStore:         ds.Seal(),
		ExistingAddresses: deployment.NewMemoryAddressBook(),
	}

	proposalCtx, err := mcmsanalyzer.NewDefaultProposalContext(env)
	require.NoError(t, err)

	tests := []struct {
		name             string
		timelockProposal string
		signers          map[mcmstypes.ChainSelector][]common.Address
		assertion        func(*testing.T, string, error)
	}{
		{
			name:             "Sui proposal with valid transaction",
			timelockProposal: timelockProposalSui,
			signers: map[mcmstypes.ChainSelector][]common.Address{
				mcmstypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					common.HexToAddress("0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"),
				},
			},
			assertion: func(t *testing.T, gotUpf string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, upfProposalSui, gotUpf)
			},
		},
		{
			name:             "Sui proposal with unknown module",
			timelockProposal: timelockProposalSuiUnknownModule,
			signers: map[mcmstypes.ChainSelector][]common.Address{
				mcmstypes.ChainSelector(chainsel.SUI_TESTNET.Selector): {
					common.HexToAddress("0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"),
				},
			},
			assertion: func(t *testing.T, gotUpf string, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, upfProposalSuiUnknownModule, gotUpf)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timelockProposal, err := mcms.NewTimelockProposal(strings.NewReader(tt.timelockProposal))
			require.NoError(t, err)
			mcmProposal := convertTimelockProposal(t.Context(), t, timelockProposal)

			got, err := UpfConvertTimelockProposal(t.Context(), proposalCtx, env, timelockProposal, mcmProposal, tt.signers)

			tt.assertion(t, got, err)
		})
	}
}

func TestUpfConvertTimelockProposalWithTon(t *testing.T) {
	t.Parallel()
	ds := datastore.NewMemoryDataStore()

	// ---- TON: testnet
	dsAddContract(t, ds, chainsel.TON_TESTNET.Selector, "EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8", "MCMS 1.0.0")

	env := deployment.Environment{
		DataStore:         ds.Seal(),
		ExistingAddresses: deployment.NewMemoryAddressBook(),
	}

	proposalCtx, err := mcmsanalyzer.NewDefaultProposalContext(env)
	require.NoError(t, err)

	tests := []struct {
		name             string
		timelockProposal string
		signers          map[mcmstypes.ChainSelector][]common.Address
		assertion        func(*testing.T, string, error)
	}{
		{
			name:             "TON proposal with GrantRole transaction",
			timelockProposal: timelockProposalTon,
			signers: map[mcmstypes.ChainSelector][]common.Address{
				mcmstypes.ChainSelector(chainsel.TON_TESTNET.Selector): {
					common.HexToAddress("0xA5D5B0B844c8f11B61F28AC98BBA84dEA9b80953"),
				},
			},
			assertion: func(t *testing.T, gotUpf string, err error) {
				t.Helper()
				require.NoError(t, err)
				// Verify it contains TON-specific content
				require.Contains(t, gotUpf, "chainFamily: ton")
				require.Contains(t, gotUpf, "chainName: ton-testnet")
				require.Contains(t, gotUpf, "msigAddress: EQADa3W6G0nSiTV4a6euRA42fU9QxSEnb-WeDpcrtWzA2jM8")
				require.Contains(t, gotUpf, "contractType: RBACTimelock")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			timelockProposal, err := mcms.NewTimelockProposal(strings.NewReader(tt.timelockProposal))
			require.NoError(t, err)
			mcmProposal := convertTimelockProposal(t.Context(), t, timelockProposal)

			got, err := UpfConvertTimelockProposal(t.Context(), proposalCtx, env, timelockProposal, mcmProposal, tt.signers)

			tt.assertion(t, got, err)
		})
	}
}

func TestIsTimelockBatchFunction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		functionName string
		want         bool
	}{
		// EVM
		{
			name:         "EVM scheduleBatch",
			functionName: "function scheduleBatch((address,uint256,bytes)[] calls, bytes32 predecessor, bytes32 salt, uint256 delay) returns()",
			want:         true,
		},
		{
			name:         "EVM bypasserExecuteBatch",
			functionName: "function bypasserExecuteBatch((address,uint256,bytes)[] calls) payable returns()",
			want:         true,
		},
		// Solana
		{
			name:         "Solana ScheduleBatch",
			functionName: "ScheduleBatch",
			want:         true,
		},
		{
			name:         "Solana BypasserExecuteBatch",
			functionName: "BypasserExecuteBatch",
			want:         true,
		},
		// Sui
		{
			name:         "Sui timelock_schedule_batch",
			functionName: "mcms::timelock_schedule_batch",
			want:         true,
		},
		{
			name:         "Sui timelock_bypasser_execute_batch",
			functionName: "mcms::timelock_bypasser_execute_batch",
			want:         true,
		},
		// Aptos
		{
			name:         "Aptos timelock_schedule_batch",
			functionName: "package::module::timelock_schedule_batch",
			want:         true,
		},
		{
			name:         "Aptos timelock_bypasser_execute_batch",
			functionName: "package::module::timelock_bypasser_execute_batch",
			want:         true,
		},
		// TON
		{
			name:         "TON ScheduleBatch",
			functionName: "com.chainlink.ton.mcms.RBACTimelock::ScheduleBatch(0x12345678)",
			want:         true,
		},
		{
			name:         "TON BypasserExecuteBatch",
			functionName: "com.chainlink.ton.mcms.RBACTimelock::BypasserExecuteBatch(0xabcdef)",
			want:         true,
		},
		// Non-matching
		{
			name:         "unrelated function",
			functionName: "function transfer(address to, uint256 amount) returns(bool)",
			want:         false,
		},
		{
			name:         "empty string",
			functionName: "",
			want:         false,
		},
		{
			name:         "partial match without colon",
			functionName: "timelock_schedule_batch",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isTimelockBatchFunction(tt.functionName)
			require.Equal(t, tt.want, got)
		})
	}
}

// ----- helpers -----

func dsAddContract(t *testing.T, ds datastore.MutableDataStore, chain uint64, addr, typeAndVersion string) {
	t.Helper()

	tv := deployment.MustTypeAndVersionFromString(typeAndVersion)
	storeAddr := addr
	if strings.HasPrefix(addr, "0x") {
		storeAddr = addr
	}
	ref := datastore.AddressRef{
		ChainSelector: chain,
		Address:       storeAddr,
		Type:          datastore.ContractType(tv.Type),
		Version:       &tv.Version,
		// Use address+type as a unique Qualifier (avoids clashes)
		Qualifier: fmt.Sprintf("%s-%s", addr, tv.Type),
	}
	if !tv.Labels.IsEmpty() {
		ref.Labels = datastore.NewLabelSet(tv.Labels.List()...)
	}

	require.NoError(t, ds.Addresses().Add(ref))
}
