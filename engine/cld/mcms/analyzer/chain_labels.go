package analyzer

import (
	"fmt"

	chainutils "github.com/smartcontractkit/chainlink-deployments-framework/chain/utils"
)

func FormatChainLabel(chainSelector uint64, chainName string) string {
	if chainName != "" {
		return fmt.Sprintf("%s (%d)", chainName, chainSelector)
	}

	return fmt.Sprintf("chain-%d", chainSelector)
}

func ResolveChainLabel(chainSelector uint64) string {
	info, _ := chainutils.ChainInfo(chainSelector)
	return FormatChainLabel(chainSelector, info.ChainName)
}
