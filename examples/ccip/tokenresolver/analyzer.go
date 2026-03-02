package tokenresolver

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
)

const (
	AnalyzerID = "ccip.token_pool.token_metadata"

	AnnotationSymbol   = "ccip.token.symbol"
	AnnotationDecimals = "ccip.token.decimals"
	AnnotationAddress  = "ccip.token.address"
)

var tokenPoolContractTypes = map[string]struct{}{
	"LockReleaseTokenPool":      {},
	"BurnMintTokenPool":         {},
	"BurnFromMintTokenPool":     {},
	"BurnWithFromMintTokenPool": {},
	"TokenPool":                 {},
}

type tokenMeta struct {
	symbol   string
	decimals uint8
	address  common.Address
}

// TokenMetadataAnalyzer resolves ERC20 token metadata.
type TokenMetadataAnalyzer struct {
	cache map[common.Address]tokenMeta
}

var _ analyzer.CallAnalyzer = (*TokenMetadataAnalyzer)(nil)

func (a *TokenMetadataAnalyzer) ID() string             { return AnalyzerID }
func (a *TokenMetadataAnalyzer) Dependencies() []string { return nil }

func (a *TokenMetadataAnalyzer) CanAnalyze(
	_ context.Context,
	_ analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call decoder.DecodedCall,
) bool {
	_, ok := tokenPoolContractTypes[call.ContractType()]

	return ok
}

func (a *TokenMetadataAnalyzer) Analyze(
	ctx context.Context,
	req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call decoder.DecodedCall,
) (annotation.Annotations, error) {
	chainSel := req.AnalyzerContext.BatchOperation().ChainSelector()
	evmChain, ok := req.ExecutionContext.BlockChains().EVMChains()[chainSel]
	if !ok {
		return nil, fmt.Errorf("EVM chain not found for selector %d", chainSel)
	}

	poolAddress := common.HexToAddress(call.To())

	meta, err := a.resolve(ctx, poolAddress, evmChain.Client)
	if err != nil {
		return nil, err
	}

	return annotation.Annotations{
		annotation.New(AnnotationSymbol, "string", meta.symbol),
		annotation.New(AnnotationDecimals, "uint8", meta.decimals),
		annotation.New(AnnotationAddress, "string", meta.address.Hex()),
	}, nil
}

func (a *TokenMetadataAnalyzer) resolve(
	ctx context.Context,
	poolAddress common.Address,
	backend bind.ContractCaller,
) (tokenMeta, error) {
	if a.cache == nil {
		a.cache = make(map[common.Address]tokenMeta)
	}

	if m, ok := a.cache[poolAddress]; ok {
		return m, nil
	}

	poolCaller, err := token_pool.NewTokenPoolCaller(poolAddress, backend)
	if err != nil {
		return tokenMeta{}, fmt.Errorf("create token pool caller for %s: %w", poolAddress, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}

	tokenAddr, err := poolCaller.GetToken(callOpts)
	if err != nil {
		return tokenMeta{}, fmt.Errorf("get token for pool %s: %w", poolAddress, err)
	}

	decimals, err := poolCaller.GetTokenDecimals(callOpts)
	if err != nil {
		return tokenMeta{}, fmt.Errorf("get decimals for pool %s: %w", poolAddress, err)
	}

	symbol := resolveSymbol(callOpts, tokenAddr, backend)

	m := tokenMeta{symbol: symbol, decimals: decimals, address: tokenAddr}
	a.cache[poolAddress] = m

	return m, nil
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
