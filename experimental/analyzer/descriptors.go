package analyzer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// Descriptor is a unit of decoded proposal. Calling Describe on it returns human-readable representation of its content.
// Some implementations are recursive (arrays, structs) and require attention to formatting.
type Descriptor interface {
	Describe(ctx *DescriptorContext) string
}

// DescriptorContext is a storage for context that may need to Descriptor during its description.
// Refer to BytesAndAddressAnalyzer and ChainSelectorAnalyzer for usage examples
type DescriptorContext struct {
	Ctx map[string]any
}

func ContextGet[T any](ctx *DescriptorContext, key string) (T, error) {
	ctxElemRaw, ok := ctx.Ctx[key]
	if !ok {
		return *new(T), fmt.Errorf("context element %s not found", key)
	}
	ctxElem, ok := ctxElemRaw.(T)
	if !ok {
		return *new(T), fmt.Errorf("context element %s type mismatch (expected: %T, was: %T)", key, ctxElem, ctxElemRaw)
	}

	return ctxElem, nil
}

func NewDescriptorContext(addresses deployment.AddressesByChain) *DescriptorContext {
	return &DescriptorContext{
		Ctx: map[string]any{
			"AddressesByChain": addresses,
		},
	}
}

type NamedDescriptor struct {
	Name  string
	Value Descriptor
}

func (n NamedDescriptor) Describe(context *DescriptorContext) string {
	return fmt.Sprintf("%s: %s", n.Name, n.Value.Describe(context))
}

type ArrayDescriptor struct {
	Elements []Descriptor
}

func (a ArrayDescriptor) Describe(context *DescriptorContext) string {
	indented := false
	elementsDescribed := make([]string, 0, len(a.Elements))
	for _, arg := range a.Elements {
		argDescribed := arg.Describe(context)
		indented = indented || strings.Contains(argDescribed, Indent)
		elementsDescribed = append(elementsDescribed, argDescribed)
	}
	description := strings.Builder{}
	if indented {
		// Write each element in new line + indentation
		description.WriteString("[\n")
		for i, elem := range elementsDescribed {
			description.WriteString(indentString(elem))
			if i < len(a.Elements)-1 {
				description.WriteString(",\n")
			}
		}
		description.WriteString("\n]")
	} else {
		// Write elements in one line
		description.WriteString("[")
		for i, elem := range elementsDescribed {
			description.WriteString(elem)
			if i < len(a.Elements)-1 {
				description.WriteString(",")
			}
		}
		description.WriteString("]")
	}

	return description.String()
}

type StructDescriptor struct {
	Fields []NamedDescriptor
}

func (s StructDescriptor) Describe(context *DescriptorContext) string {
	description := strings.Builder{}
	if len(s.Fields) >= MinStructFieldsForPrettyFormat {
		// Pretty format struct with indentation
		description.WriteString("{\n")
		for _, arg := range s.Fields {
			description.WriteString(indentString(arg.Describe(context)))
			description.WriteString("\n")
		}
		description.WriteString("}")
	} else {
		// Struct in one line
		description.WriteString("{ ")
		for i, arg := range s.Fields {
			description.WriteString(arg.Describe(context))
			if i < len(s.Fields)-1 {
				description.WriteString(", ")
			}
		}
		description.WriteString(" }")
	}

	return description.String()
}

type SimpleDescriptor struct {
	Value string
}

func (s SimpleDescriptor) Describe(_ *DescriptorContext) string {
	return s.Value
}

type ChainSelectorDescriptor struct {
	Value uint64
}

func (c ChainSelectorDescriptor) Describe(_ *DescriptorContext) string {
	chainName, err := GetChainNameBySelector(c.Value)
	if err != nil || chainName == "" {
		return fmt.Sprintf("%d (<chain unknown>)", c.Value)
	}

	return fmt.Sprintf("%d (%s)", c.Value, chainName)
}

type BytesDescriptor struct {
	Value []byte
}

func (a BytesDescriptor) Describe(_ *DescriptorContext) string {
	return hexutil.Encode(a.Value)
}

type AddressDescriptor struct {
	Value string
}

// Annotation returns only the annotation if known, otherwise "".
func (a AddressDescriptor) Annotation(ctx *DescriptorContext) string {
	addresses, err := ContextGet[deployment.AddressesByChain](ctx, "AddressesByChain")
	if err != nil {
		return ""
	}
	for chainSel, addresses := range addresses {
		chainName, err := GetChainNameBySelector(chainSel)
		if err != nil || chainName == "" {
			chainName = strconv.FormatUint(chainSel, 10)
		}
		typeAndVersion, ok := addresses[a.Value]
		if ok {
			return fmt.Sprintf("address of %s from %s", typeAndVersion.String(), chainName)
		}
	}

	return ""
}

func (a AddressDescriptor) Describe(_ *DescriptorContext) string {
	return a.Value
}
