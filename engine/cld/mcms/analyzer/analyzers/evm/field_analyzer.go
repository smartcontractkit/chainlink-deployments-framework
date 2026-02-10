package evm

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
	expanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer/pointer"
)

const evmFieldAnalyzerID = "evm-field-analyzer"

// EVMFieldParameterAnalyzer is a built-in ParameterAnalyzer that uses the
// experimental decoder's FieldValue analysis. It walks the display tree produced
// by the decoder and converts semantic findings (chain selectors, addresses, etc.)
type EVMFieldParameterAnalyzer struct{}

var _ analyzer.ParameterAnalyzer = &EVMFieldParameterAnalyzer{}

func (a *EVMFieldParameterAnalyzer) ID() string             { return evmFieldAnalyzerID }
func (a *EVMFieldParameterAnalyzer) Dependencies() []string { return nil }

func (a *EVMFieldParameterAnalyzer) Matches(_ context.Context, _ analyzer.AnalyzerRequest, param analyzer.DecodedParameter) bool {
	_, ok := param.DisplayValue.(expanalyzer.FieldValue)
	return ok
}

func (a *EVMFieldParameterAnalyzer) Analyze(_ context.Context, req analyzer.AnalyzerRequest, param analyzer.DecodedParameter) (analyzer.Annotations, error) {
	fv, ok := param.DisplayValue.(expanalyzer.FieldValue)
	if !ok {
		return nil, nil
	}

	fieldCtx := buildFieldContext(req)

	var annotations analyzer.Annotations
	walkFieldValue(param.Name, fv, fieldCtx, &annotations)

	return annotations, nil
}

func walkFieldValue(path string, fv expanalyzer.FieldValue, fieldCtx *expanalyzer.FieldContext, out *analyzer.Annotations) {
	switch v := fv.(type) {
	case expanalyzer.ChainSelectorField:
		chainName, err := expanalyzer.GetChainNameBySelector(v.GetValue())
		if err != nil {
			chainName = fmt.Sprintf("unknown(%d)", v.GetValue())
		}

		*out = append(*out, analyzer.NewAnnotationWithAnalyzer(
			"chain_selector", path, chainName, evmFieldAnalyzerID,
		))

	case expanalyzer.AddressField:
		value := v.GetValue()
		if fieldCtx != nil {
			if annotation := v.Annotation(fieldCtx); annotation != "" {
				value = annotation
			}
		}

		*out = append(*out, analyzer.NewAnnotationWithAnalyzer(
			"address", path, value, evmFieldAnalyzerID,
		))

	case expanalyzer.StructField:
		for _, field := range v.GetFields() {
			if field.Value != nil {
				walkFieldValue(path+"."+field.Name, field.Value, fieldCtx, out)
			}
		}

	case expanalyzer.ArrayField:
		for i, elem := range v.GetElements() {
			if elem != nil {
				walkFieldValue(fmt.Sprintf("%s[%d]", path, i), elem, fieldCtx, out)
			}
		}

	case expanalyzer.BytesField:
		if v.GetLength() > 0 {
			*out = append(*out, analyzer.NewAnnotationWithAnalyzer(
				"bytes", path, fmt.Sprintf("0x%x (%d bytes)", v.GetValue(), v.GetLength()), evmFieldAnalyzerID,
			))
		}
	}
}

func buildFieldContext(req analyzer.AnalyzerRequest) *expanalyzer.FieldContext {
	if req.Execution == nil || req.Execution.Env.DataStore == nil {
		return expanalyzer.NewFieldContext(nil)
	}

	addresses, err := req.Execution.Env.DataStore.Addresses().Fetch()
	if err != nil {
		return expanalyzer.NewFieldContext(nil)
	}

	addressesByChain := make(deployment.AddressesByChain)
	for _, addr := range addresses {
		chainAddrs, exists := addressesByChain[addr.ChainSelector]
		if !exists {
			chainAddrs = make(map[string]deployment.TypeAndVersion)
			addressesByChain[addr.ChainSelector] = chainAddrs
		}

		chainAddrs[addr.Address] = deployment.TypeAndVersion{
			Type:    deployment.ContractType(addr.Type),
			Version: pointer.DerefOrEmpty(addr.Version),
		}
	}

	return expanalyzer.NewFieldContext(addressesByChain)
}
