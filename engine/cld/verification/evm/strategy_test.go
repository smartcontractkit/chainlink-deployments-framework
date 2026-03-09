package evm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVerificationStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chainID  uint64
		expected VerificationStrategy
	}{
		{"Ethereum mainnet", 1, StrategyEtherscan},
		{"Sepolia", 11155111, StrategyEtherscan},
		{"Arbitrum One", 42161, StrategyEtherscan},
		{"Base", 8453, StrategyEtherscan},
		{"Optimism", 10, StrategyEtherscan},
		{"Polygon", 137, StrategyEtherscan},
		{"Metis", 1088, StrategyBlockscout},
		{"Zora", 7777777, StrategyBlockscout},
		{"Avalanche C-Chain", 43114, StrategyRoutescan},
		{"BSC", 56, StrategyEtherscan},
		{"X Layer", 196, StrategyOkLink},
		{"BTR mainnet", 200901, StrategyBtrScan},
		{"CoreDAO", 1116, StrategyCoreDAO},
		{"Blast Sepolia", 168587773, StrategyEtherscan},
		{"Linea", 59144, StrategyEtherscan},
		{"zkSync Era", 324, StrategyEtherscan},
		{"Mantle", 5000, StrategyEtherscan},
		{"Scroll", 534352, StrategyEtherscan},
		{"Sourcify chain", 295, StrategySourcify},
		{"L2Scan chain", 4200, StrategyL2Scan},
		{"SocialScan chain", 688688, StrategySocialScan},
		{"Unknown chain", 999999999, StrategyUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := GetVerificationStrategy(tt.chainID)
			require.Equal(t, tt.expected, got, "chainID=%d", tt.chainID)
		})
	}
}

func TestIsChainSupportedOnEtherscanV2(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnEtherscanV2(1))
	require.True(t, IsChainSupportedOnEtherscanV2(137))
	require.False(t, IsChainSupportedOnEtherscanV2(999999999))
}

func TestIsChainSupportedOnBlockscout(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnBlockscout(1088))
	require.True(t, IsChainSupportedOnBlockscout(7777777))
	require.False(t, IsChainSupportedOnBlockscout(1))
}

func TestIsChainSupportedOnRouteScan(t *testing.T) {
	t.Parallel()

	nt, ok := IsChainSupportedOnRouteScan(43114)
	require.True(t, ok)
	require.Equal(t, "mainnet", nt)

	nt, ok = IsChainSupportedOnRouteScan(43113)
	require.True(t, ok)
	require.Equal(t, "testnet", nt)

	_, ok = IsChainSupportedOnRouteScan(1)
	require.False(t, ok)
}

func TestIsChainSupportedOnSourcify(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnSourcify(295))
	require.False(t, IsChainSupportedOnSourcify(1))
}

func TestIsChainSupportedOnOkLink(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnOkLink(196))
	require.False(t, IsChainSupportedOnOkLink(1))
}

func TestGetOkLinkShortName(t *testing.T) {
	t.Parallel()

	name, ok := GetOkLinkShortName(196)
	require.True(t, ok)
	require.Equal(t, "XLAYER", name)

	_, ok = GetOkLinkShortName(1)
	require.False(t, ok)
}

func TestIsChainSupportedOnBtrScan(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnBtrScan(200901))
	require.False(t, IsChainSupportedOnBtrScan(1))
}

func TestIsChainSupportedOnCoreDAO(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnCoreDAO(1116))
	require.False(t, IsChainSupportedOnCoreDAO(1))
}

func TestIsChainSupportedOnL2Scan(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnL2Scan(4200))
	require.False(t, IsChainSupportedOnL2Scan(1))
}

func TestIsChainSupportedOnSocialScanV2(t *testing.T) {
	t.Parallel()

	require.True(t, IsChainSupportedOnSocialScanV2(688688))
	require.False(t, IsChainSupportedOnSocialScanV2(1))
}
