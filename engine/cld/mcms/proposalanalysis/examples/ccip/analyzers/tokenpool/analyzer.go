package tokenpool

import (
	"context"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip/analyzers/tokenresolver"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/format"
)

const AnalyzerID = "ccip.token_pool.apply_chain_updates"

type ChainUpdatesAnalyzer struct{}

var _ analyzer.CallAnalyzer = (*ChainUpdatesAnalyzer)(nil)

func (a *ChainUpdatesAnalyzer) ID() string             { return AnalyzerID }
func (a *ChainUpdatesAnalyzer) Dependencies() []string { return []string{tokenresolver.AnalyzerID} }

func (a *ChainUpdatesAnalyzer) CanAnalyze(
	_ context.Context,
	_ analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call analyzer.DecodedCall,
) bool {
	return ccip.IsTokenPoolContract(call.ContractType()) && call.Name() == "applyChainUpdates"
}

func (a *ChainUpdatesAnalyzer) Analyze(
	ctx context.Context,
	req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call analyzer.DecodedCall,
) (analyzer.Annotations, error) {
	chainSel := req.AnalyzerContext.BatchOperation().ChainSelector()
	evmChains := req.ExecutionContext.BlockChains().EVMChains()
	evmChain, ok := evmChains[chainSel]
	if !ok {
		return nil, fmt.Errorf("EVM chain not found for selector %d", chainSel)
	}

	chainsToAdd, selectorsToRemove, err := extractChainUpdateParams(call)
	if err != nil {
		return nil, fmt.Errorf("extract chain update params: %w", err)
	}
	if len(chainsToAdd) == 0 && len(selectorsToRemove) == 0 {
		return analyzer.Annotations{
			analyzer.SeverityAnnotation(analyzer.SeverityWarning),
			analyzer.NewAnnotation("ccip.warning", "string", "no chain updates provided"),
		}, nil
	}

	if !common.IsHexAddress(call.To()) {
		return nil, fmt.Errorf("invalid pool address %q", call.To())
	}

	poolAddress := common.HexToAddress(call.To())
	poolCaller, err := token_pool.NewTokenPoolCaller(poolAddress, evmChain.Client)
	if err != nil {
		return nil, fmt.Errorf("create token pool caller for %s: %w", poolAddress, err)
	}

	callOpts := &bind.CallOpts{Context: ctx}
	currentChains, err := poolCaller.GetSupportedChains(callOpts)
	if err != nil {
		return nil, fmt.Errorf("get supported chains for %s: %w", poolAddress, err)
	}

	tokenSymbol, decimals := readTokenMetadata(req.DependencyAnnotationStore)

	var anns analyzer.Annotations

	for _, update := range chainsToAdd {
		remoteLabel := resolveChainLabel(update.RemoteChainSelector)
		chainExists := slices.Contains(currentChains, update.RemoteChainSelector)

		if !chainExists {
			anns = append(anns, analyzer.NewAnnotation("ccip.chain_update", "chain_update",
				ccip.ChainUpdateValue{RemoteChainSelector: update.RemoteChainSelector, Label: remoteLabel + " added"}))
		} else {
			anns = append(anns,
				analyzer.SeverityAnnotation(analyzer.SeverityWarning),
				analyzer.NewAnnotation("ccip.chain_update", "chain_update",
					ccip.ChainUpdateValue{RemoteChainSelector: update.RemoteChainSelector, Label: remoteLabel + " already enabled"}),
			)
		}

		current, rerr := poolCaller.GetCurrentOutboundRateLimiterState(callOpts, update.RemoteChainSelector)
		if rerr != nil {
			return nil, fmt.Errorf("get outbound rate limiter for %s: %w", remoteLabel, rerr)
		}

		anns = append(anns, compareRateLimiterConfig(
			current, update.OutboundRateLimiterConfig, decimals, "outbound to "+remoteLabel, tokenSymbol,
		)...)

		currentIn, ierr := poolCaller.GetCurrentInboundRateLimiterState(callOpts, update.RemoteChainSelector)
		if ierr != nil {
			return nil, fmt.Errorf("get inbound rate limiter for %s: %w", remoteLabel, ierr)
		}

		anns = append(anns, compareRateLimiterConfig(
			currentIn, update.InboundRateLimiterConfig, decimals, "inbound from "+remoteLabel, tokenSymbol,
		)...)
	}

	for _, sel := range selectorsToRemove {
		remoteLabel := resolveChainLabel(sel)
		chainExists := slices.Contains(currentChains, sel)

		if chainExists {
			anns = append(anns, analyzer.NewAnnotation("ccip.chain_update", "chain_update",
				ccip.ChainUpdateValue{RemoteChainSelector: sel, Label: remoteLabel + " removed"}))
		} else {
			anns = append(anns,
				analyzer.SeverityAnnotation(analyzer.SeverityWarning),
				analyzer.NewAnnotation("ccip.chain_update", "chain_update",
					ccip.ChainUpdateValue{RemoteChainSelector: sel, Label: remoteLabel + " already disabled"}),
			)
		}
	}

	return anns, nil
}

func compareRateLimiterConfig(
	current token_pool.RateLimiterTokenBucket,
	proposed token_pool.RateLimiterConfig,
	decimals uint8,
	direction string,
	tokenSymbol string,
) analyzer.Annotations {
	var anns analyzer.Annotations

	if current.IsEnabled != proposed.IsEnabled {
		action := "disabled"
		if proposed.IsEnabled {
			action = "enabled"
		}

		anns = append(anns, analyzer.NewAnnotation(
			"ccip.rate_limiter", "string",
			fmt.Sprintf("%s: rate limiter %s", direction, action),
		))
	}

	curCap := orZero(current.Capacity)
	propCap := orZero(proposed.Capacity)

	if curCap.Cmp(propCap) != 0 {
		anns = append(anns, analyzer.DiffAnnotation(
			direction+" capacity",
			formatRichAmount(curCap, decimals, tokenSymbol),
			formatRichAmount(propCap, decimals, tokenSymbol),
			"",
		))

		if propCap.Sign() == 0 {
			anns = append(anns, analyzer.SeverityAnnotation(analyzer.SeverityWarning))
			anns = append(anns, analyzer.RiskAnnotation(analyzer.RiskHigh))
		}
	}

	curRate := orZero(current.Rate)
	propRate := orZero(proposed.Rate)

	if curRate.Cmp(propRate) != 0 {
		anns = append(anns, analyzer.DiffAnnotation(
			direction+" rate",
			formatRichAmount(curRate, decimals, tokenSymbol),
			formatRichAmount(propRate, decimals, tokenSymbol),
			"",
		))

		if propRate.Sign() == 0 {
			anns = append(anns, analyzer.SeverityAnnotation(analyzer.SeverityWarning))
			anns = append(anns, analyzer.RiskAnnotation(analyzer.RiskHigh))
		}
	}

	return anns
}

func readTokenMetadata(store analyzer.DependencyAnnotationStore) (string, uint8) {
	var symbol string
	var decimals uint8

	if anns := store.Filter(analyzer.ByAnnotationAnalyzer(tokenresolver.AnalyzerID), analyzer.ByAnnotationName(tokenresolver.AnnotationSymbol)); len(anns) > 0 {
		symbol, _ = anns[0].Value().(string)
	}

	if anns := store.Filter(analyzer.ByAnnotationAnalyzer(tokenresolver.AnalyzerID), analyzer.ByAnnotationName(tokenresolver.AnnotationDecimals)); len(anns) > 0 {
		decimals, _ = anns[0].Value().(uint8)
	}

	return symbol, decimals
}

func resolveChainLabel(chainSelector uint64) string {
	return fmt.Sprintf("%s (%d)", format.ResolveChainName(chainSelector), chainSelector)
}

func formatRichAmount(amount *big.Int, decimals uint8, tokenSymbol string) string {
	if amount == nil || amount.Sign() == 0 {
		return "0"
	}

	if tokenSymbol == "" && decimals == 0 {
		return format.CommaGroupBigInt(amount) + " (decimals=unknown)"
	}

	human := format.FormatTokenAmount(amount, decimals)

	return fmt.Sprintf("%s %s (%s, decimals=%d)", human, tokenSymbol, format.CommaGroupBigInt(amount), decimals)
}

func orZero(n *big.Int) *big.Int {
	if n == nil {
		return big.NewInt(0)
	}

	return n
}
