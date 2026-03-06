package evm

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

// GetVerificationStrategy returns the verification strategy for the given chain ID.
func GetVerificationStrategy(chainID uint64) VerificationStrategy {
	if IsChainSupportedOnEtherscanV2(chainID) {
		return StrategyEtherscan
	}
	if _, ok := IsChainSupportedOnRouteScan(chainID); ok {
		return StrategyRoutescan
	}
	if IsChainSupportedOnSourcify(chainID) {
		return StrategySourcify
	}
	if IsChainSupportedOnOkLink(chainID) {
		return StrategyOkLink
	}
	if IsChainSupportedOnBtrScan(chainID) {
		return StrategyBtrScan
	}
	if IsChainSupportedOnCoreDAO(chainID) {
		return StrategyCoreDAO
	}
	if IsChainSupportedOnBlockscout(chainID) {
		return StrategyBlockscout
	}
	if IsChainSupportedOnL2Scan(chainID) {
		return StrategyL2Scan
	}
	if IsChainSupportedOnSocialScanV2(chainID) {
		return StrategySocialScan
	}

	return StrategyUnknown
}
