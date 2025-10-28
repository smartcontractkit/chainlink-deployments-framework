package analyzer

import (
	"fmt"
	"strconv"
	"strings"

	solana "github.com/gagliardetto/solana-go"
	"github.com/goccy/go-yaml"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
)

// FieldValue is an interface for different types of field values.
// it is used in analyzers to represent different types of data.
type FieldValue interface {
	GetType() string
}

// FieldContext is a storage for context that may be needed for field representations.
// Refer to BytesAndAddressFieldAnalyzer and ChainSelectorFieldAnalyzer for usage examples
type FieldContext struct {
	Ctx map[string]any
}

func NewFieldContext(addresses deployment.AddressesByChain) *FieldContext {
	return &FieldContext{
		Ctx: map[string]any{
			"AddressesByChain": addresses,
		},
	}
}

func FieldContextGet[T any](fieldCtx *FieldContext, key string) (T, error) {
	ctxElemRaw, ok := fieldCtx.Ctx[key]
	if !ok {
		return *new(T), fmt.Errorf("context element %s not found", key)
	}
	ctxElem, ok := ctxElemRaw.(T)
	if !ok {
		return *new(T), fmt.Errorf("context element %s type mismatch (expected: %T, was: %T)", key, ctxElem, ctxElemRaw)
	}

	return ctxElem, nil
}

type NamedField struct {
	Name  string
	Value FieldValue
}

func (n NamedField) GetType() string {
	return "NamedField"
}

func (n NamedField) GetName() string {
	return n.Name
}

func (n NamedField) GetValue() FieldValue {
	return n.Value
}

type ArrayField struct {
	Elements []FieldValue
}

func (a ArrayField) GetType() string {
	return "ArrayField"
}

func (a ArrayField) GetElements() []FieldValue {
	return a.Elements
}

func (a ArrayField) GetLength() int {
	return len(a.Elements)
}

type StructField struct {
	Fields []NamedField
}

func (s StructField) GetType() string {
	return "StructField"
}

func (s StructField) GetFields() []NamedField {
	return s.Fields
}

func (s StructField) GetFieldCount() int {
	return len(s.Fields)
}

type SimpleField struct {
	Value string
}

func (s SimpleField) GetType() string {
	return "SimpleField"
}

func (s SimpleField) GetValue() string {
	return s.Value
}

type ChainSelectorField struct {
	Value uint64
}

func (c ChainSelectorField) GetType() string {
	return "ChainSelectorField"
}

func (c ChainSelectorField) GetValue() uint64 {
	return c.Value
}

type BytesField struct {
	Value []byte
}

func (b BytesField) GetType() string {
	return "BytesField"
}

func (b BytesField) GetValue() []byte {
	return b.Value
}

func (b BytesField) GetLength() int {
	return len(b.Value)
}

type AddressField struct {
	Value string
}

func (a AddressField) GetType() string {
	return "AddressField"
}

func (a AddressField) GetValue() string {
	return a.Value
}

// Annotation returns only the annotation if known, otherwise "".
func (a AddressField) Annotation(ctx *FieldContext) string {
	addresses, err := FieldContextGet[deployment.AddressesByChain](ctx, "AddressesByChain")
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

// YamlField for Solana-specific YAML output
// TODO: could also be used for the UPF format when making that format implement the Renderer interface.
type YamlField struct {
	Value any
}

func (y YamlField) GetType() string {
	return "YamlField"
}

// MarshalYAML marshals the yaml field using custom marshallers
func (y YamlField) MarshalYAML() ([]byte, error) {
	return yaml.MarshalWithOptions(y.Value, customMarshallers...)
}

func (y YamlField) GetValue() string {
	// If the value is already a string, return it directly
	if str, ok := y.Value.(string); ok {
		return str
	}

	wrappedValue := map[string]any{"__key__": y.Value}
	marshaled, err := yaml.MarshalWithOptions(wrappedValue, customMarshallers...)
	if err != nil {
		return fmt.Sprintf("%#v", y.Value)
	}
	out := string(marshaled)[8:]
	out = strings.TrimSpace(out)
	out = strings.ReplaceAll(out, "\n", "")

	return out
}

var customMarshallers = []yaml.EncodeOption{
	yaml.CustomMarshaler(func(value []byte) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value []uint8) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value [16]uint8) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value [20]uint8) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value [32]byte) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value [32]uint8) ([]byte, error) { return fmt.Appendf(nil, "0x%x", value), nil }),
	yaml.CustomMarshaler(func(value solana.AccountMeta) ([]byte, error) {
		out := fmt.Appendf(nil, "%-46s", value.PublicKey)
		if value.IsWritable {
			out = fmt.Appendf(out, " [writable]")
		}
		if value.IsSigner {
			out = fmt.Appendf(out, " [signer]")
		}

		return out, nil
	}),
}
