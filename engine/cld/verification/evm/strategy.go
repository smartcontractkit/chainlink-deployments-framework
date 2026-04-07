package evm

import (
	"strings"

	cfgnet "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/network"
)

// Legacy chain-ID tables (previously embedded in CLDF) — kept here as documentation only.
// Configure the matching block_explorer in CLD instead of restoring allowlists in code.
//
// Routescan API path segment was chainID as a decimal string except:
//   - testnet 9746 (Plasma) → "9746_5"
//   (other testnet/mainnet IDs matched their decimal string: 21000001, 3636, 80069, 43113, 21000000, 3637, 80094, 9745, 43114)
//
// OKLink explorer slug: 196 → XLAYER
// SocialScan v2 slug: 688688 → pharos-testnet
// Sourcify (strategy only if type/url set in CLD): 295, 296, 2020, 2021, 42431
// BtrScan: 200901 | CoreDAO: 1116, 1115 | L2Scan: 4200, 686868, 223
// Blockscout (verifier no longer chain-gated): 1868, 98866, 7777777, 1088, 1329, 43111, 42793, 185, 57073, 1135, 177, 60808, 1750, 47763, 34443, 5330, 592, 30, 2818, 2810, 36888, 26888, 99999, 36900
// Etherscan v2: previously a large allowlist in CLDF; see https://docs.etherscan.io/etherscan-v2/supported-chains
//
// Slug meaning by type:
//   - Routescan: optional override for the /evm/{segment}/ path when segment ≠ decimal chain ID (e.g. 9746_5).
//   - OKLink / SocialScan: required chain identifier for their APIs (not an override of numeric chain ID).

// VerificationStrategy identifies which block explorer API to use for a chain.
type VerificationStrategy int

const (
	StrategyUnknown VerificationStrategy = iota
	StrategyEtherscan
	StrategyOkLink
	StrategyBlockscout
	StrategyRoutescan
	StrategySourcify
	StrategyBtrScan
	StrategyCoreDAO
	StrategyL2Scan
	StrategySocialScan
)

// GetVerificationStrategyForNetwork returns the verifier strategy from domain network config only
// (block_explorer.type and related fields). There is no chain-ID allowlist in CLDF; CLD must set
// block_explorer appropriately for each network.
func GetVerificationStrategyForNetwork(n cfgnet.Network) VerificationStrategy {
	ex := n.BlockExplorer
	t := strings.ToLower(strings.TrimSpace(ex.Type))
	switch t {
	case "etherscan", "snowtrace":
		if !hasAPIKey(ex) && strings.TrimSpace(ex.URL) == "" {
			return StrategyUnknown
		}

		return StrategyEtherscan
	case "wemixscan", "zircuit":
		if !hasAPIKey(ex) || strings.TrimSpace(ex.URL) == "" {
			return StrategyUnknown
		}

		return StrategyEtherscan
	case "blockscout":
		if strings.TrimSpace(ex.URL) == "" {
			return StrategyUnknown
		}

		return StrategyBlockscout
	case "routescan":
		if routescanNetworkType(n) == "" {
			return StrategyUnknown
		}

		return StrategyRoutescan
	case "oklink":
		if strings.TrimSpace(ex.Slug) == "" {
			return StrategyUnknown
		}

		return StrategyOkLink
	case "sourcify":
		if strings.TrimSpace(ex.URL) == "" {
			return StrategyUnknown
		}

		return StrategySourcify
	case "btrscan", "coredao", "l2scan":
		if strings.TrimSpace(ex.URL) == "" {
			return StrategyUnknown
		}
		switch t {
		case "btrscan":
			return StrategyBtrScan
		case "coredao":
			return StrategyCoreDAO
		default:
			return StrategyL2Scan
		}
	case "socialscan":
		if strings.TrimSpace(ex.Slug) == "" {
			return StrategyUnknown
		}

		return StrategySocialScan
	default:
		return StrategyUnknown
	}
}

func hasAPIKey(ex cfgnet.BlockExplorer) bool {
	return strings.TrimSpace(ex.APIKey) != ""
}

// routescanNetworkType returns the Routescan API network segment from CLD network.type (mainnet | testnet).
func routescanNetworkType(n cfgnet.Network) string {
	switch n.Type {
	case cfgnet.NetworkTypeMainnet:
		return "mainnet"
	case cfgnet.NetworkTypeTestnet:
		return "testnet"
	default:
		return ""
	}
}
