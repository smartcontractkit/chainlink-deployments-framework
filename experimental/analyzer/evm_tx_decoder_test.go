package analyzer

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// from: github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/v1_6_0/rmn_home
var rmnHomeABI = "[{\"type\":\"function\",\"name\":\"acceptOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getActiveDigest\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAllConfigs\",\"inputs\":[],\"outputs\":[{\"name\":\"activeConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.VersionedConfig\",\"components\":[{\"name\":\"version\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"staticConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.StaticConfig\",\"components\":[{\"name\":\"nodes\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.Node[]\",\"components\":[{\"name\":\"peerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"offchainPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"candidateConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.VersionedConfig\",\"components\":[{\"name\":\"version\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"staticConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.StaticConfig\",\"components\":[{\"name\":\"nodes\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.Node[]\",\"components\":[{\"name\":\"peerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"offchainPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCandidateDigest\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getConfig\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"versionedConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.VersionedConfig\",\"components\":[{\"name\":\"version\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"staticConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.StaticConfig\",\"components\":[{\"name\":\"nodes\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.Node[]\",\"components\":[{\"name\":\"peerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"offchainPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}]},{\"name\":\"ok\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getConfigDigests\",\"inputs\":[],\"outputs\":[{\"name\":\"activeConfigDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"candidateConfigDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"promoteCandidateAndRevokeActive\",\"inputs\":[{\"name\":\"digestToPromote\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"digestToRevoke\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"revokeCandidate\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setCandidate\",\"inputs\":[{\"name\":\"staticConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.StaticConfig\",\"components\":[{\"name\":\"nodes\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.Node[]\",\"components\":[{\"name\":\"peerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"offchainPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"digestToOverwrite\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"newConfigDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setDynamicConfig\",\"inputs\":[{\"name\":\"newDynamicConfig\",\"type\":\"tuple\",\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"currentDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"typeAndVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"ActiveConfigRevoked\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"CandidateConfigRevoked\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigPromoted\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ConfigSet\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"version\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"},{\"name\":\"staticConfig\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structRMNHome.StaticConfig\",\"components\":[{\"name\":\"nodes\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.Node[]\",\"components\":[{\"name\":\"peerId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"offchainPublicKey\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DynamicConfigSet\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"dynamicConfig\",\"type\":\"tuple\",\"indexed\":false,\"internalType\":\"structRMNHome.DynamicConfig\",\"components\":[{\"name\":\"sourceChains\",\"type\":\"tuple[]\",\"internalType\":\"structRMNHome.SourceChain[]\",\"components\":[{\"name\":\"chainSelector\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"fObserve\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"observerNodesBitmap\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"offchainConfig\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferRequested\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"CannotTransferToSelf\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ConfigDigestMismatch\",\"inputs\":[{\"name\":\"expectedConfigDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"gotConfigDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"DigestNotFound\",\"inputs\":[{\"name\":\"configDigest\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"DuplicateOffchainPublicKey\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DuplicatePeerId\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"DuplicateSourceChain\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"MustBeProposedOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NoOpStateTransitionNotAllowed\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotEnoughObservers\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OnlyCallableByOwner\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OutOfBoundsNodesLength\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OutOfBoundsObserverNodeIndex\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnerCannotBeZero\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"RevokingZeroDigestNotAllowed\",\"inputs\":[]}]"

func Test_AnalyzeRmnHomeSetConfig(t *testing.T) {
	t.Parallel()
	dataEncoded := "EY26xQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAaAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAASAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA+mHAtKPceHWHsUU/gZ3gbQUSqxoUZSI6A1wsufMtT1Ds0lEhXpCRE0bKF15QNbgZoIwngeB5DppZ27p+FxzwtFJl4urMXKidA5eY8JSHVPtI1h4ZiTOwbcUx4ju2t4QG2jZ8/J045hVKKkjqbrOPTnFXdd4sYe0Egs4TMSMiShZgAHkr6j6Bhgz3+4p0ob33+C3inZ/H6HxUrzaTVNBUGFa+O4yMWpkJ/FppF/cGzqRqFrEWePBy5HGnhxR8MH8IQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3kG6T8nZGtkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAzPCjGiIfPJsAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMEYRtq/7p2oAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAHAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	data := make([]byte, base64.StdEncoding.DecodedLen(len(dataEncoded)))
	_, err := base64.StdEncoding.Decode(data, []byte(dataEncoded))
	require.NoError(t, err)
	_abi, err := abi.JSON(strings.NewReader(rmnHomeABI))
	require.NoError(t, err)
	decoder := NewTxCallDecoder(nil)
	analyzeResult, err := decoder.Decode(common.Address{}.Hex(), &_abi, data)
	require.NoError(t, err)
	assert.Equal(t, _abi.Methods["setCandidate"].String(), analyzeResult.Method)
	assert.Equal(t, "staticConfig", analyzeResult.Inputs[0].Name)
	assert.Equal(t, StructField{
		Fields: []NamedField{
			{
				Name: "Nodes",
				Value: ArrayField{
					Elements: []FieldValue{
						StructField{
							Fields: []NamedField{
								{
									Name:  "PeerId",
									Value: BytesField{Value: hexutil.MustDecode("0xe98702d28f71e1d61ec514fe067781b4144aac68519488e80d70b2e7ccb53d43")},
								},
								{
									Name:  "OffchainPublicKey",
									Value: BytesField{Value: hexutil.MustDecode("0xb34944857a42444d1b285d7940d6e06682309e0781e43a69676ee9f85c73c2d1")},
								},
							},
						},
						StructField{
							Fields: []NamedField{
								{
									Name:  "PeerId",
									Value: BytesField{Value: hexutil.MustDecode("0x49978bab3172a2740e5e63c2521d53ed2358786624cec1b714c788eedade101b")},
								},
								{
									Name:  "OffchainPublicKey",
									Value: BytesField{Value: hexutil.MustDecode("0x68d9f3f274e3985528a923a9bace3d39c55dd778b187b4120b384cc48c892859")},
								},
							},
						},
						StructField{
							Fields: []NamedField{
								{
									Name:  "PeerId",
									Value: BytesField{Value: hexutil.MustDecode("0x8001e4afa8fa061833dfee29d286f7dfe0b78a767f1fa1f152bcda4d53415061")},
								},
								{
									Name:  "OffchainPublicKey",
									Value: BytesField{Value: hexutil.MustDecode("0x5af8ee32316a6427f169a45fdc1b3a91a85ac459e3c1cb91c69e1c51f0c1fc21")},
								},
							},
						},
					},
				},
			},
			{
				Name:  "OffchainConfig",
				Value: BytesField{Value: hexutil.MustDecode("0x")},
			},
		},
	}, analyzeResult.Inputs[0].Value)
	assert.Equal(t, "dynamicConfig", analyzeResult.Inputs[1].Name)
	assert.Equal(t, StructField{
		Fields: []NamedField{
			{
				Name: "SourceChains",
				Value: ArrayField{
					Elements: []FieldValue{
						StructField{
							Fields: []NamedField{
								{
									Name:  "ChainSelector",
									Value: ChainSelectorField{Value: 16015286601757825753},
								},
								{
									Name:  "FObserve",
									Value: SimpleField{Value: "1"},
								},
								{
									Name:  "ObserverNodesBitmap",
									Value: SimpleField{Value: "7"},
								},
							},
						},
						StructField{
							Fields: []NamedField{
								{
									Name:  "ChainSelector",
									Value: ChainSelectorField{Value: 14767482510784806043},
								},
								{
									Name:  "FObserve",
									Value: SimpleField{Value: "1"},
								},
								{
									Name:  "ObserverNodesBitmap",
									Value: SimpleField{Value: "7"},
								},
							},
						},
						StructField{
							Fields: []NamedField{
								{
									Name:  "ChainSelector",
									Value: ChainSelectorField{Value: 3478487238524512106},
								},
								{
									Name:  "FObserve",
									Value: SimpleField{Value: "1"},
								},
								{
									Name:  "ObserverNodesBitmap",
									Value: SimpleField{Value: "7"},
								},
							},
						},
					},
				},
			},
			{
				Name:  "OffchainConfig",
				Value: BytesField{Value: hexutil.MustDecode("0x")},
			},
		},
	}, analyzeResult.Inputs[1].Value)
	assert.Equal(t, "digestToOverwrite", analyzeResult.Inputs[2].Name)
	assert.Equal(t, BytesField{Value: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000")}, analyzeResult.Inputs[2].Value)
	assert.Equal(t, "newConfigDigest", analyzeResult.Outputs[0].Name)
	assert.Equal(t, BytesField{Value: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000060")}, analyzeResult.Outputs[0].Value)
}
