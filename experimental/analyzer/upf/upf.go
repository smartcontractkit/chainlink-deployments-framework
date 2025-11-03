package upf

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-yaml"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/smartcontractkit/mcms"
	mcmsaptossdk "github.com/smartcontractkit/mcms/sdk/aptos"
	mcmssuisdk "github.com/smartcontractkit/mcms/sdk/sui"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	mcmsanalyzer "github.com/smartcontractkit/chainlink-deployments-framework/experimental/analyzer"
)

// UpfConvertTimelockProposal converts a TimelockProposal to a UPF proposal format.
func UpfConvertTimelockProposal(
	proposalCtx mcmsanalyzer.ProposalContext,
	timelockProposal *mcms.TimelockProposal,
	mcmProposal *mcms.Proposal,
	signers map[mcmstypes.ChainSelector][]common.Address,
) (string, error) {
	upfProposal, err := mcmsProposalToUpfProposal(proposalCtx, mcmProposal, timelockProposal.TimelockAddresses, signers)
	if err != nil {
		return "", fmt.Errorf("failed to convert proposal to upf format: %w", err)
	}

	decodedBatches, err := batchOperationsToUpfDecodedCalls(proposalCtx, timelockProposal.Operations)
	if err != nil {
		return "", fmt.Errorf("failed to describe batch operations: %w", err)
	}

	decodedBatchesIndex := 0
	for _, batch := range upfProposal.Transactions {
		if batch.Metadata == nil || batch.Metadata.DecodedCalldata == nil {
			continue
		}
		if batch.Metadata.ContractType == "RBACTimelock" &&
			(batch.Metadata.DecodedCalldata.FunctionName == "function scheduleBatch((address,uint256,bytes)[] calls, bytes32 predecessor, bytes32 salt, uint256 delay) returns()" ||
				batch.Metadata.DecodedCalldata.FunctionName == "function bypasserExecuteBatch((address,uint256,bytes)[] calls) payable returns()" ||
				batch.Metadata.DecodedCalldata.FunctionName == "BypasserExecuteBatch" ||
				batch.Metadata.DecodedCalldata.FunctionName == "ScheduleBatch") {
			batch.Metadata.DecodedCalldata.FunctionArgs["calls"] = decodedBatches[decodedBatchesIndex]
			decodedBatchesIndex++
		}
	}

	marshaled, err := yaml.MarshalWithOptions(upfProposal, upfYamlMarshallers()...)
	if err != nil {
		return "", fmt.Errorf("failed to marshal UPF proposal: %w", err)
	}

	return "---\n" + string(marshaled), nil
}

// UpfConvertProposal converts a standard MCMS proposal to a UPF proposal format.
func UpfConvertProposal(
	proposalCtx mcmsanalyzer.ProposalContext,
	proposal *mcms.Proposal,
	signers map[mcmstypes.ChainSelector][]common.Address,
) (string, error) {
	upfProposal, err := mcmsProposalToUpfProposal(proposalCtx, proposal, map[mcmstypes.ChainSelector]string{}, signers)
	if err != nil {
		return "", fmt.Errorf("failed to convert proposal to upf format: %w", err)
	}

	marshaled, err := yaml.MarshalWithOptions(upfProposal, upfYamlMarshallers()...)
	if err != nil {
		return "", fmt.Errorf("failed to marshal UPF proposal: %w", err)
	}

	return "---\n" + string(marshaled), nil
}

func mcmsProposalToUpfProposal(
	proposalCtx mcmsanalyzer.ProposalContext,
	proposal *mcms.Proposal,
	timelockAddresses map[mcmstypes.ChainSelector]string,
	signers map[mcmstypes.ChainSelector][]common.Address,
) (UPFProposal, error) {
	merkleTree, err := proposal.MerkleTree()
	if err != nil {
		return UPFProposal{}, fmt.Errorf("failed to get merkle tree: %w", err)
	}
	signingHash, err := proposal.SigningHash()
	if err != nil {
		return UPFProposal{}, fmt.Errorf("failed to get signing hash: %w", err)
	}
	signingMessage, err := proposal.SigningMessage()
	if err != nil {
		return UPFProposal{}, fmt.Errorf("failed to get signing message: %w", err)
	}

	transactions := make([]Transaction, len(proposal.Operations))
	for i, op := range proposal.Operations {
		transactions[i], err = mcmsOperationToUpfTransaction(proposalCtx, op, i, proposal, timelockAddresses)
		if err != nil {
			return UPFProposal{}, fmt.Errorf("failed to convert mcms operation to upf transaction %d: %w", i, err)
		}
	}
	signersStr := make(map[mcmstypes.ChainSelector][]string, len(signers))
	for chainSelector, addresses := range signers {
		signersStr[chainSelector] = []string{}
		for _, addr := range addresses {
			signersStr[chainSelector] = append(signersStr[chainSelector], addr.Hex())
		}
	}
	upfProposal := UPFProposal{
		ProposalHash: signingHash.Hex(),
		MsigType:     "mcms",
		Signers:      signersStr,
		McmsParams: &McmsParams{
			ValidUntil:           proposal.ValidUntil,
			OverridePreviousRoot: proposal.OverridePreviousRoot,
			MerkleRoot:           merkleTree.Root.Hex(),
			ASCIIProposalHash:    asciiHash(signingMessage),
		},
		Transactions: transactions,
	}

	return upfProposal, nil
}

func mcmsOperationToUpfTransaction(
	proposalCtx mcmsanalyzer.ProposalContext, mcmsOp mcmstypes.Operation, idx int, proposal *mcms.Proposal,
	timelockAddresses map[mcmstypes.ChainSelector]string,
) (Transaction, error) {
	chainFamily, err := chainsel.GetSelectorFamily(uint64(mcmsOp.ChainSelector))
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to get chain family for selector %v: %w", mcmsOp.ChainSelector, err)
	}
	chainID, err := chainsel.GetChainIDFromSelector(uint64(mcmsOp.ChainSelector))
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to get chain id for selector %v: %w", mcmsOp.ChainSelector, err)
	}
	chainDetails, err := chainsel.GetChainDetailsByChainIDAndFamily(chainID, chainFamily)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to get chain details for selector %v: %w", mcmsOp.ChainSelector, err)
	}
	chainMetadata := proposal.ChainMetadata[mcmsOp.ChainSelector]
	opCount := calculateOpCount(chainMetadata.StartingOpCount, idx, proposal.Operations)

	additionalFields := struct{ Value int64 }{}
	err = json.Unmarshal(mcmsOp.Transaction.AdditionalFields, &additionalFields)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to unmarshal \"additionalFields\" attribute: %w", err)
	}

	analyzeResult, _, err := analyzeTransaction(proposalCtx, mcmsOp)
	if err != nil {
		return Transaction{}, err
	}

	encodedTransactionData, err := encodeTransactionData(mcmsOp)
	if err != nil {
		return Transaction{}, fmt.Errorf("failed to encode transaction data: %w", err)
	}

	upfFunctionArgs := make(DecodedCalldataFunctionArgs, len(analyzeResult.Inputs))
	for _, arg := range analyzeResult.Inputs {
		upfFunctionArgs[arg.Name] = arg.Value
	}

	return Transaction{
		Index:           idx,
		ChainFamily:     chainFamily,
		ChainID:         chainID,
		ChainName:       chainDetails.ChainName,
		ChainShortName:  chainDetails.ChainName,
		MsigAddress:     chainMetadata.MCMAddress,
		TimelockAddress: timelockAddresses[mcmsOp.ChainSelector],
		To:              mcmsOp.Transaction.To,
		Value:           additionalFields.Value,
		Data:            encodedTransactionData,
		TxNonce:         opCount,
		Metadata: &Metadata{
			DecodedCalldata: &DecodedCallData{
				FunctionName: analyzeResult.Method,
				FunctionArgs: upfFunctionArgs,
			},
			Comment:      "",
			ContractType: mcmsOp.Transaction.ContractType,
		},
	}, nil
}

func asciiHash(data [32]byte) rawBytes {
	var sb strings.Builder

	for _, byteVal := range data {
		// Check if the byte is a printable ASCII character (32 to 126 are printable)
		if byteVal >= 32 && byteVal <= 126 {
			sb.WriteByte(byteVal)
		} else if byteVal >= 9 && byteVal <= 13 {
			// 09 to 13 are tab, newline, vertical tab, form feed, and carriage return.
			// Just print them as a space character
			sb.WriteByte(' ')
		} else {
			// Append the escaped hexadecimal representation of the byte
			fmt.Fprintf(&sb, "\\x%02x", byteVal)
		}
	}

	return rawBytes(sb.String())
}

func calculateOpCount(opCount uint64, opIndex int, operations []mcmstypes.Operation) uint64 {
	chainSelector := operations[opIndex].ChainSelector
	for i, op := range operations {
		if i == opIndex {
			break
		}
		if op.ChainSelector == chainSelector {
			opCount += 1
		}
	}

	return opCount
}

func encodeTransactionData(mcmsOp mcmstypes.Operation) (string, error) {
	chainFamily, err := chainsel.GetSelectorFamily(uint64(mcmsOp.ChainSelector))
	if err != nil {
		return "", fmt.Errorf("failed to get chain family for selector %v: %w", mcmsOp.ChainSelector, err)
	}

	switch chainFamily {
	case chainsel.FamilySolana:
		return base64.StdEncoding.EncodeToString(mcmsOp.Transaction.Data), nil
	case chainsel.FamilyAptos:
		return base64.StdEncoding.EncodeToString(mcmsOp.Transaction.Data), nil
	case chainsel.FamilySui:
		return base64.StdEncoding.EncodeToString(mcmsOp.Transaction.Data), nil
	default:
		return "0x" + hex.EncodeToString(mcmsOp.Transaction.Data), nil
	}
}

func batchOperationsToUpfDecodedCalls(ctx mcmsanalyzer.ProposalContext, batches []mcmstypes.BatchOperation) ([][]*DecodedInnerCall, error) {
	decodedCalls := make([][]*DecodedInnerCall, len(batches))

	for batchIdx, batch := range batches {
		chainSel := uint64(batch.ChainSelector)
		family, err := chainsel.GetSelectorFamily(chainSel)
		if err != nil {
			return nil, err
		}

		decodedCalls[batchIdx] = make([]*DecodedInnerCall, len(batch.Transactions))

		switch family {
		case chainsel.FamilyEVM:
			describedTxs, err := mcmsanalyzer.AnalyzeEVMTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, tx := range describedTxs {
				decodedCalls[batchIdx][callIdx] = &DecodedInnerCall{
					To:   tx.Address,
					Data: cldDecodedCallToUpfDecodedCallData(tx),
				}
			}

		case chainsel.FamilySolana:
			describedTxs, err := mcmsanalyzer.AnalyzeSolanaTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, tx := range describedTxs {
				decodedCalls[batchIdx][callIdx] = &DecodedInnerCall{
					To:   tx.Address,
					Data: cldDecodedCallToUpfDecodedCallData(tx),
				}
			}

		case chainsel.FamilyAptos:
			describedTxs, err := mcmsanalyzer.AnalyzeAptosTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, tx := range describedTxs {
				decodedCalls[batchIdx][callIdx] = &DecodedInnerCall{
					To:   tx.Address,
					Data: cldDecodedCallToUpfDecodedCallData(tx),
				}
			}

		case chainsel.FamilySui:
			describedTxs, err := mcmsanalyzer.AnalyzeSuiTransactions(ctx, chainSel, batch.Transactions)
			if err != nil {
				return nil, err
			}
			for callIdx, tx := range describedTxs {
				decodedCalls[batchIdx][callIdx] = &DecodedInnerCall{
					To:   tx.Address,
					Data: cldDecodedCallToUpfDecodedCallData(tx),
				}
			}

		default:
			for callIdx, mcmsTx := range batch.Transactions {
				decodedCalls[batchIdx][callIdx] = &DecodedInnerCall{
					To:   mcmsTx.To,
					Data: &DecodedCallData{FunctionName: family + " transaction decoding is not supported"},
				}
			}
		}
	}

	return decodedCalls, nil
}

func cldDecodedCallToUpfDecodedCallData(cldDecodedCall *mcmsanalyzer.DecodedCall) *DecodedCallData {
	upfFunctionArgs := make(DecodedCalldataFunctionArgs, len(cldDecodedCall.Inputs))
	for _, arg := range cldDecodedCall.Inputs {
		upfFunctionArgs[arg.Name] = arg.Value
	}

	return &DecodedCallData{FunctionName: cldDecodedCall.Method, FunctionArgs: upfFunctionArgs}
}

func analyzeTransaction(
	proposalCtx mcmsanalyzer.ProposalContext, mcmsOp mcmstypes.Operation,
) (*mcmsanalyzer.DecodedCall, string, error) {
	chainFamily, err := chainsel.GetSelectorFamily(uint64(mcmsOp.ChainSelector))
	if err != nil {
		return nil, "", fmt.Errorf("failed to get chain family for selector %v: %w", mcmsOp.ChainSelector, err)
	}

	switch chainFamily {
	case chainsel.FamilyEVM:
		decoder := mcmsanalyzer.NewTxCallDecoder(nil) // FIXME: reuse instance
		analyzeResult, _, abi, err := mcmsanalyzer.AnalyzeEVMTransaction(proposalCtx, decoder, uint64(mcmsOp.ChainSelector), mcmsOp.Transaction)
		if err != nil {
			return nil, "", fmt.Errorf("failed to analyze EVM transaction: %w", err)
		}

		analyzeResult.Inputs = describeInputs(proposalCtx, analyzeResult.Inputs, mcmsOp.ChainSelector)

		return analyzeResult, abi, nil

	case chainsel.FamilySolana:
		analyzeResult, err := mcmsanalyzer.AnalyzeSolanaTransaction(proposalCtx, uint64(mcmsOp.ChainSelector), mcmsOp.Transaction)
		if err != nil {
			return nil, "", err
		}

		return analyzeResult, "", nil

	case chainsel.FamilyAptos:
		decoder := mcmsaptossdk.NewDecoder()
		analyzeResult, err := mcmsanalyzer.AnalyzeAptosTransaction(proposalCtx, decoder, uint64(mcmsOp.ChainSelector), mcmsOp.Transaction)
		if err != nil {
			return nil, "", err
		}

		return analyzeResult, "", nil

	case chainsel.FamilySui:
		decoder := mcmssuisdk.NewDecoder()
		analyzeResult, err := mcmsanalyzer.AnalyzeSuiTransaction(proposalCtx, decoder, uint64(mcmsOp.ChainSelector), mcmsOp.Transaction)
		if err != nil {
			return nil, "", err
		}

		return analyzeResult, "", nil

	default:
		return nil, "", fmt.Errorf("unsupported chain family: %s", chainFamily)
	}
}

func upfYamlMarshallers() []yaml.EncodeOption {
	// This function provides custom YAML marshaling for UPF format.
	// It could be refactored into a dedicated Renderer object to improve code organization
	// and make the marshaling logic more reusable across different output formats.
	return []yaml.EncodeOption{
		yaml.CustomMarshaler(func(arg mcmsanalyzer.SimpleField) ([]byte, error) {
			return yaml.Marshal(arg.Value)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.NamedField) ([]byte, error) {
			return yaml.MarshalWithOptions(map[string]any{arg.Name: arg.Value}, upfYamlMarshallers()...)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.ArrayField) ([]byte, error) {
			return yaml.MarshalWithOptions(arg.Elements, upfYamlMarshallers()...)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.StructField) ([]byte, error) {
			argMap := map[string]any{}
			for _, field := range arg.Fields {
				argMap[field.Name] = field.Value
			}

			return yaml.MarshalWithOptions(argMap, upfYamlMarshallers()...)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.ChainSelectorField) ([]byte, error) {
			return yaml.Marshal(arg.Value)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.AddressField) ([]byte, error) {
			return yaml.Marshal(arg.Value)
		}),
		yaml.CustomMarshaler(func(arg mcmsanalyzer.BytesField) ([]byte, error) {
			return yaml.Marshal(fmt.Sprintf("0x%x", arg.Value))
		}),
		yaml.CustomMarshaler(func(field mcmsanalyzer.YamlField) ([]byte, error) {
			return field.MarshalYAML()
		}),
	}
}

func describeInputs(
	_ mcmsanalyzer.ProposalContext, inputs []mcmsanalyzer.NamedField, _ mcmstypes.ChainSelector,
) []mcmsanalyzer.NamedField {
	renderer := mcmsanalyzer.NewTextRenderer()
	describedInputs := make([]mcmsanalyzer.NamedField, len(inputs))

	for i, arg := range inputs {
		// Use RenderFieldValue to get just the value without the "name: " prefix
		// This is more efficient and less fragile than calling RenderField and stripping the prefix
		valueStr := renderer.RenderFieldValue(arg.Value)
		describedInputs[i] = mcmsanalyzer.NamedField{
			Name:  arg.Name,
			Value: mcmsanalyzer.SimpleField{Value: valueStr},
		}
	}

	return describedInputs
}
