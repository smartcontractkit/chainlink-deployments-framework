package tokenpool

import (
	"context"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	chainutils "github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotation"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/analyzer/annotationstore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/decoder"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/proposalanalysis/examples/ccip/tokenresolver"
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
	call decoder.DecodedCall,
) bool {
	return ccip.IsTokenPoolContract(call.ContractType()) && call.Name() == "applyChainUpdates"
}

func (a *ChainUpdatesAnalyzer) Analyze(
	ctx context.Context,
	req analyzer.AnalyzeRequest[analyzer.CallAnalyzerContext],
	call decoder.DecodedCall,
) (annotation.Annotations, error) {
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
		return annotation.Annotations{
			annotation.SeverityAnnotation(annotation.SeverityWarning),
			annotation.New("ccip.warning", "string", "no chain updates provided"),
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

	var anns annotation.Annotations

	for _, update := range chainsToAdd {
		remoteLabel := resolveChainLabel(update.RemoteChainSelector)
		chainExists := slices.Contains(currentChains, update.RemoteChainSelector)

		if !chainExists {
			anns = append(anns, annotation.New("ccip.chain_update", "string", remoteLabel+" added"))
		} else {
			anns = append(anns,
				annotation.SeverityAnnotation(annotation.SeverityWarning),
				annotation.New("ccip.chain_update", "string", remoteLabel+" already enabled"),
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
			anns = append(anns, annotation.New("ccip.chain_update", "string", remoteLabel+" removed"))
		} else {
			anns = append(anns,
				annotation.SeverityAnnotation(annotation.SeverityWarning),
				annotation.New("ccip.chain_update", "string", remoteLabel+" already disabled"),
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
) annotation.Annotations {
	var anns annotation.Annotations

	if current.IsEnabled != proposed.IsEnabled {
		action := "disabled"
		if proposed.IsEnabled {
			action = "enabled"
		}

		anns = append(anns, annotation.New(
			"ccip.rate_limiter", "string",
			fmt.Sprintf("%s: rate limiter %s", direction, action),
		))
	}

	curCap := orZero(current.Capacity)
	propCap := orZero(proposed.Capacity)

	if curCap.Cmp(propCap) != 0 {
		anns = append(anns, annotation.DiffAnnotation(
			direction+" capacity",
			formatRichAmount(curCap, decimals, tokenSymbol),
			formatRichAmount(propCap, decimals, tokenSymbol),
			"",
		))

		if propCap.Sign() == 0 {
			anns = append(anns, annotation.SeverityAnnotation(annotation.SeverityWarning))
			anns = append(anns, annotation.RiskAnnotation(annotation.RiskHigh))
		}
	}

	curRate := orZero(current.Rate)
	propRate := orZero(proposed.Rate)

	if curRate.Cmp(propRate) != 0 {
		anns = append(anns, annotation.DiffAnnotation(
			direction+" rate",
			formatRichAmount(curRate, decimals, tokenSymbol),
			formatRichAmount(propRate, decimals, tokenSymbol),
			"",
		))

		if propRate.Sign() == 0 {
			anns = append(anns, annotation.SeverityAnnotation(annotation.SeverityWarning))
			anns = append(anns, annotation.RiskAnnotation(annotation.RiskHigh))
		}
	}

	return anns
}

func readTokenMetadata(store annotationstore.DependencyAnnotationStore) (string, uint8) {
	var symbol string
	var decimals uint8

	if anns := store.Filter(annotationstore.ByAnalyzer(tokenresolver.AnalyzerID), annotationstore.ByName(tokenresolver.AnnotationSymbol)); len(anns) > 0 {
		symbol, _ = anns[0].Value().(string)
	}

	if anns := store.Filter(annotationstore.ByAnalyzer(tokenresolver.AnalyzerID), annotationstore.ByName(tokenresolver.AnnotationDecimals)); len(anns) > 0 {
		decimals, _ = anns[0].Value().(uint8)
	}

	return symbol, decimals
}

func resolveChainLabel(chainSelector uint64) string {
	info, err := chainutils.ChainInfo(chainSelector)
	if err != nil {
		return fmt.Sprintf("chain-%d", chainSelector)
	}

	return fmt.Sprintf("%s (%d)", info.ChainName, chainSelector)
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
