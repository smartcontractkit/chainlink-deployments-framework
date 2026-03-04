package tokenresolver

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
)

const (
	AnalyzerID = "ccip.token_pool.token_metadata"

	AnnotationSymbol   = "ccip.token.symbol"
	AnnotationDecimals = "ccip.token.decimals"
	AnnotationAddress  = "ccip.token.address"
)

// TokenMetadataAnalyzer resolves ERC20 token metadata (symbol, decimals,
// address) for any token pool call.
type TokenMetadataAnalyzer struct{}

var _ analyzer.CallAnalyzer = (*TokenMetadataAnalyzer)(nil)

func (a *TokenMetadataAnalyzer) ID() string             { return AnalyzerID }
func (a *TokenMetadataAnalyzer) Dependencies() []string { return nil }

func (a *TokenMetadataAnalyzer) CanAnalyze(
	_ context.Context,
	_ analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call analyzer.DecodedCall,
) bool {
	return ccip.IsTokenPoolContract(call.ContractType())
}

func (a *TokenMetadataAnalyzer) Analyze(
	ctx context.Context,
	req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call analyzer.DecodedCall,
) (analyzer.Annotations, error) {
	if !common.IsHexAddress(call.To()) {
		return nil, fmt.Errorf("invalid pool address %q", call.To())
	}

	chainSel := req.AnalyzerContext.BatchOperation().ChainSelector()
	evmChain, ok := req.ExecutionContext.BlockChains().EVMChains()[chainSel]
	if !ok {
		return nil, fmt.Errorf("EVM chain not found for selector %d", chainSel)
	}

	poolAddress := common.HexToAddress(call.To())

	poolCaller, err := token_pool.NewTokenPoolCaller(poolAddress, evmChain.Client)
	if err != nil {
		return nil, fmt.Errorf("create token pool caller for %s: %w", poolAddress, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}

	tokenAddr, err := poolCaller.GetToken(callOpts)
	if err != nil {
		return nil, fmt.Errorf("get token address from pool %s: %w", poolAddress, err)
	}

	decimals, err := poolCaller.GetTokenDecimals(callOpts)
	if err != nil {
		return nil, fmt.Errorf("get token decimals from pool %s: %w", poolAddress, err)
	}

	symbol := resolveSymbol(callOpts, tokenAddr, evmChain.Client)

	return analyzer.Annotations{
		analyzer.NewAnnotation(AnnotationSymbol, "string", symbol),
		analyzer.NewAnnotation(AnnotationDecimals, "uint8", decimals),
		analyzer.NewAnnotation(AnnotationAddress, "string", tokenAddr.Hex()),
	}, nil
}

func resolveSymbol(
	callOpts *bind.CallOpts,
	tokenAddress common.Address,
	backend bind.ContractCaller,
) string {
	caller := newERC20Caller(tokenAddress, backend)
	symbol, err := caller.symbol(callOpts)
	if err != nil || symbol == "" {
		return tokenAddress.Hex()[:10]
	}

	return symbol
}
