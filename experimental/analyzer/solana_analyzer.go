package analyzer

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	binary "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/goccy/go-yaml"
	mcmssolanasdk "github.com/smartcontractkit/mcms/sdk/solana"
	mcmstypes "github.com/smartcontractkit/mcms/types"
)

func AnalyzeSolanaTransactions(
	ctx ProposalContext, chainSelector uint64, txs []mcmstypes.Transaction,
) ([]*DecodedCall, error) {
	decodedTxs := make([]*DecodedCall, len(txs))
	for i, op := range txs {
		analyzedTransaction, err := AnalyzeSolanaTransaction(ctx, chainSelector, op)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze solana transaction %d: %w", i, err)
		}

		decodedTxs[i] = analyzedTransaction
	}

	return decodedTxs, nil
}

func AnalyzeSolanaTransaction(
	ctx ProposalContext, chainSelector uint64, mcmsTx mcmstypes.Transaction,
) (*DecodedCall, error) {
	decodedTx := &DecodedCall{
		Inputs:  []NamedDescriptor{},
		Outputs: []NamedDescriptor{},
	}
	solReg := ctx.GetSolanaDecoderRegistry()
	if solReg == nil {
		return nil, errors.New("solana decoder registry is not available")
	}
	decodeFn, err := solReg.GetSolanaInstructionDecoderByAddress(chainSelector, mcmsTx.To)
	if err != nil {
		return nil, fmt.Errorf("failed to get solana program: %w", err)
	}

	var solanaAdditionalFields mcmssolanasdk.AdditionalFields
	err = json.Unmarshal(mcmsTx.AdditionalFields, &solanaAdditionalFields)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal additional fields: %w", err)
	}

	instruction, err := decodeFn(solanaAdditionalFields.Accounts, mcmsTx.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode solana instruction: %w", err)
	}

	decodedTx.Address = mcmsTx.To
	decodedTx.Method = instruction.Name()
	decodedTx.Inputs = instruction.Inputs()

	return decodedTx, nil
}

type DecodeInstructionFn func(accounts []*solana.AccountMeta, data []byte) (AnchorInstruction, error)

func DIFn[T any](fn func(accounts []*solana.AccountMeta, data []byte) (T, error)) DecodeInstructionFn {
	return func(accounts []*solana.AccountMeta, data []byte) (AnchorInstruction, error) {
		instruction, err := fn(accounts, data)
		if err != nil {
			return nil, err
		}

		return &anchorInstructionWrapper{anchorInstruction: instruction}, nil
	}
}

type anchorInstructionWrapper struct {
	anchorInstruction any
}

type AnchorInstruction interface {
	solana.Instruction
	Name() string
	TypeID() (binary.TypeID, error)
	Impl() (any, error)
	Inputs() []NamedDescriptor
}

func (w *anchorInstructionWrapper) Name() string {
	impl, err := w.Impl()
	if err != nil {
		return "<unknown>"
	}

	implType := reflect.TypeOf(impl)
	if implType.Kind() == reflect.Ptr {
		return implType.Elem().Name()
	}

	return implType.Name()
}

func (w *anchorInstructionWrapper) baseVariant() (binary.BaseVariant, error) {
	if reflect.ValueOf(w.anchorInstruction).Kind() != reflect.Ptr {
		return binary.BaseVariant{}, errors.New("invalid type in anchor instruction (not a pointer)")
	}
	if reflect.ValueOf(w.anchorInstruction).Elem().Kind() != reflect.Struct {
		return binary.BaseVariant{}, errors.New("invalid type in anchor instruction (not a struct)")
	}
	variant := reflect.ValueOf(w.anchorInstruction).Elem().FieldByName("BaseVariant")
	if !variant.IsValid() {
		return binary.BaseVariant{}, errors.New("failed to get BaseVariant field in anchor instruction")
	}

	baseVariant, ok := variant.Convert(reflect.TypeOf(binary.BaseVariant{})).Interface().(binary.BaseVariant)
	if !ok {
		return binary.BaseVariant{}, errors.New("unable to convert BaseVariant field to binary.BaseVariant type")
	}

	return baseVariant, nil
}

func (w *anchorInstructionWrapper) TypeID() (binary.TypeID, error) {
	baseVariant, err := w.baseVariant()
	if err != nil {
		return binary.TypeID{}, err
	}

	return baseVariant.TypeID, nil
}

func (w *anchorInstructionWrapper) Impl() (any, error) {
	baseVariant, err := w.baseVariant()
	if err != nil {
		return nil, err
	}

	return baseVariant.Impl, nil
}

func (w *anchorInstructionWrapper) Inputs() []NamedDescriptor {
	impl, err := w.Impl()
	if err != nil {
		return []NamedDescriptor{{
			Name:  "error",
			Value: SimpleDescriptor{Value: err.Error()},
		}}
	}
	if reflect.ValueOf(impl).Elem().Kind() != reflect.Struct {
		return []NamedDescriptor{{
			Name:  "error",
			Value: SimpleDescriptor{Value: "unxpected BaseVariant.Impl element type (not a struct)"},
		}}
	}

	rImpl := reflect.ValueOf(impl)
	if rImpl.Kind() != reflect.Ptr {
		return []NamedDescriptor{{
			Name:  "error",
			Value: SimpleDescriptor{Value: "unxpected BaseVariant.Impl type (not a pointer)"},
		}}
	}
	if rImpl.Elem().Kind() != reflect.Struct {
		return []NamedDescriptor{{
			Name:  "error",
			Value: SimpleDescriptor{Value: "unxpected BaseVariant.Impl element type (not a struct)"},
		}}
	}
	rImpl = rImpl.Elem()

	inputs := make([]NamedDescriptor, rImpl.NumField())
	for i := range rImpl.NumField() {
		inputs[i].Name = rImpl.Type().Field(i).Name
		inputs[i].Value = YamlDescriptor{value: rImpl.Field(i).Interface()}
	}

	return inputs
}

func (w *anchorInstructionWrapper) ProgramID() solana.PublicKey {
	return w.anchorInstruction.(solana.Instruction).ProgramID()
}

func (w *anchorInstructionWrapper) Accounts() []*solana.AccountMeta {
	return w.anchorInstruction.(solana.Instruction).Accounts()
}

func (w *anchorInstructionWrapper) Data() ([]byte, error) {
	return w.anchorInstruction.(solana.Instruction).Data()
}

type YamlDescriptor struct {
	value any
}

func (y YamlDescriptor) Describe(_ *DescriptorContext) string {
	wrappedValue := map[string]any{"__key__": y.value}
	marshaled, err := yaml.MarshalWithOptions(wrappedValue, customMarshallers...)
	if err != nil {
		return fmt.Sprintf("%#v", y.value)
	}

	return strings.Trim(string(marshaled)[8:], " ") // [8:] drops the "__key__:" prefix
}

func (y YamlDescriptor) MarshalYAML() ([]byte, error) {
	return yaml.MarshalWithOptions(y.value, customMarshallers...)
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
