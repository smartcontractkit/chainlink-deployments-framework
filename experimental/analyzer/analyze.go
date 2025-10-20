package analyzer

import (
	"reflect"
	"regexp"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// Magic number constants
	MinStructFieldsForPrettyFormat = 2
	MinDataLengthForMethodID       = 4
	DefaultAnalyzersCount          = 2
)

var (
	_                  Analyzer = BytesAndAddressAnalyzer
	_                  Analyzer = ChainSelectorAnalyzer
	chainSelectorRegex          = regexp.MustCompile(`[cC]hain([sS]el)?.*$`)
)

// Analyzer is an extension point of proposal decoding.
// You can implement your own Analyzer which returns your own Descriptor instance.
type Analyzer func(argName string, argAbi *abi.Type, argVal any, analyzers []Analyzer) Descriptor

type DecodedCall struct {
	Address string
	Method  string
	Inputs  []NamedDescriptor
	Outputs []NamedDescriptor
}

// Describe renders a human-readable representation of the decoded call,
// delegating to the Markdown renderer for consistency.
func (d *DecodedCall) Describe(context *DescriptorContext) string {
	return NewMarkdownRenderer().RenderDecodedCall(d, context)
}

func BytesAndAddressAnalyzer(_ string, argAbi *abi.Type, argVal any, _ []Analyzer) Descriptor {
	if argAbi.T == abi.FixedBytesTy || argAbi.T == abi.BytesTy || argAbi.T == abi.AddressTy {
		argArrTyp := reflect.ValueOf(argVal)
		argArr := make([]byte, argArrTyp.Len())
		for i := range argArrTyp.Len() {
			argArr[i] = byte(argArrTyp.Index(i).Uint())
		}
		if argAbi.T == abi.AddressTy {
			return AddressDescriptor{Value: common.BytesToAddress(argArr).Hex()}
		}

		return BytesDescriptor{Value: argArr}
	}

	return nil
}

func ChainSelectorAnalyzer(argName string, argAbi *abi.Type, argVal any, _ []Analyzer) Descriptor {
	if argAbi.GetType().Kind() == reflect.Uint64 && chainSelectorRegex.MatchString(argName) {
		return ChainSelectorDescriptor{Value: argVal.(uint64)}
	}

	return nil
}
