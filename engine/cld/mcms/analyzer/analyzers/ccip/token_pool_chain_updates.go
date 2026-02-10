package ccip

import (
	"context"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated/latest/token_pool"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/analyzer"
)

//nolint:gosec // not credentials
const tokenPoolChainUpdatesID = "ccip.token_pool.apply_chain_updates"

type TokenPoolChainUpdatesAnalyzer struct {
	tokenSymbols map[common.Address]string
}

var _ analyzer.CallAnalyzer = &TokenPoolChainUpdatesAnalyzer{}

func (a *TokenPoolChainUpdatesAnalyzer) ID() string {
	return tokenPoolChainUpdatesID
}

func (a *TokenPoolChainUpdatesAnalyzer) Dependencies() []string {
	return nil
}

func (a *TokenPoolChainUpdatesAnalyzer) Matches(
	_ context.Context,
	_ analyzer.AnalyzerRequest,
	call analyzer.DecodedCall,
) bool {
	ct := call.ContractType()
	isTokenPool := ct == "LockReleaseTokenPool" ||
		ct == "BurnMintTokenPool" ||
		ct == "BurnFromMintTokenPool" ||
		ct == "BurnWithFromMintTokenPool" ||
		ct == "TokenPool"

	return isTokenPool && call.Name() == "applyChainUpdates"
}

func (a *TokenPoolChainUpdatesAnalyzer) Analyze(
	ctx context.Context,
	req analyzer.AnalyzerRequest,
	call analyzer.DecodedCall,
) (analyzer.Annotations, error) {
	chainSel := req.Context.BatchOperation.ChainSelector
	evmChains := req.Execution.Env.BlockChains.EVMChains()

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
			analyzer.NewAnnotation("warning", "", "No chain updates provided"),
		}, nil
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

	tokenAddress, err := poolCaller.GetToken(callOpts)
	if err != nil {
		return nil, fmt.Errorf("get token for pool %s: %w", poolAddress, err)
	}

	decimals, err := poolCaller.GetTokenDecimals(callOpts)
	if err != nil {
		return nil, fmt.Errorf("get decimals for pool %s: %w", poolAddress, err)
	}

	tokenSymbol := a.getTokenSymbol(callOpts, tokenAddress, evmChain.Client)

	var annotations analyzer.Annotations

	for _, update := range chainsToAdd {
		remoteLabel := analyzer.ResolveChainLabel(update.RemoteChainSelector)

		chainExists := slices.Contains(currentChains, update.RemoteChainSelector)
		if !chainExists {
			annotations = append(annotations, analyzer.NewAnnotation("chain update", "", remoteLabel+" added"))
		} else {
			annotations = append(annotations, analyzer.NewAnnotation("warning", "", remoteLabel+" already enabled"))
		}

		if update.OutboundRateLimiterConfig.Capacity != nil {
			current, rerr := poolCaller.GetCurrentOutboundRateLimiterState(callOpts, update.RemoteChainSelector)
			if rerr != nil {
				return nil, fmt.Errorf("get outbound rate limiter for %s: %w", remoteLabel, rerr)
			}

			annotations = append(annotations, compareRateLimiterConfig(
				current, update.OutboundRateLimiterConfig,
				decimals, "outbound to "+remoteLabel, tokenSymbol,
			)...)
		}

		if update.InboundRateLimiterConfig.Capacity != nil {
			current, rerr := poolCaller.GetCurrentInboundRateLimiterState(callOpts, update.RemoteChainSelector)
			if rerr != nil {
				return nil, fmt.Errorf("get inbound rate limiter for %s: %w", remoteLabel, rerr)
			}

			annotations = append(annotations, compareRateLimiterConfig(
				current, update.InboundRateLimiterConfig,
				decimals, "inbound from "+remoteLabel, tokenSymbol,
			)...)
		}
	}

	for _, sel := range selectorsToRemove {
		remoteLabel := analyzer.ResolveChainLabel(sel)

		chainExists := slices.Contains(currentChains, sel)
		if chainExists {
			annotations = append(annotations, analyzer.NewAnnotation("chain update", "", remoteLabel+" removed"))
		} else {
			annotations = append(annotations, analyzer.NewAnnotation("warning", "", remoteLabel+" already disabled"))
		}
	}

	return annotations, nil
}

func (a *TokenPoolChainUpdatesAnalyzer) getTokenSymbol(
	callOpts *bind.CallOpts,
	tokenAddress common.Address,
	backend bind.ContractCaller,
) string {
	if a.tokenSymbols == nil {
		a.tokenSymbols = make(map[common.Address]string)
	}

	if s, ok := a.tokenSymbols[tokenAddress]; ok {
		return s
	}

	tokenCaller := newERC20Caller(tokenAddress, backend)

	symbol, err := tokenCaller.symbol(callOpts)
	if err != nil || symbol == "" {
		symbol = tokenAddress.Hex()[:10]
	}

	a.tokenSymbols[tokenAddress] = symbol

	return symbol
}

func compareRateLimiterConfig(
	current token_pool.RateLimiterTokenBucket,
	proposed token_pool.RateLimiterConfig,
	decimals uint8,
	direction string,
	tokenName string,
) analyzer.Annotations {
	var annotations analyzer.Annotations

	if current.IsEnabled != proposed.IsEnabled {
		action := "disabled"
		if proposed.IsEnabled {
			action = "enabled"
		}

		annotations = append(annotations, analyzer.NewAnnotation("rate limiter", direction, "rate limiter "+action))
	}

	if proposed.Capacity != nil && current.Capacity.Cmp(proposed.Capacity) != 0 {
		currentFmt := FormatTokenAmount(current.Capacity, decimals)
		proposedFmt := FormatTokenAmount(proposed.Capacity, decimals)

		annotationType := "rate limiter"
		if proposedFmt == "0" {
			annotationType = "warning"
		}

		annotations = append(annotations, analyzer.NewAnnotation(annotationType, direction, fmt.Sprintf(
			"capacity %s -> %s %s tokens (%s -> %s, decimals=%d)",
			currentFmt, proposedFmt, tokenName,
			current.Capacity.String(), proposed.Capacity.String(), decimals,
		)))
	}

	if proposed.Rate != nil && current.Rate.Cmp(proposed.Rate) != 0 {
		currentFmt := FormatTokenAmount(current.Rate, decimals)
		proposedFmt := FormatTokenAmount(proposed.Rate, decimals)

		annotationType := "rate limiter"
		if proposedFmt == "0" {
			annotationType = "warning"
		}

		annotations = append(annotations, analyzer.NewAnnotation(annotationType, direction, fmt.Sprintf(
			"rate %s -> %s %s tokens (%s -> %s, decimals=%d)",
			currentFmt, proposedFmt, tokenName,
			current.Rate.String(), proposed.Rate.String(), decimals,
		)))
	}

	return annotations
}
