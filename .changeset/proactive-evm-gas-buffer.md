---
"chainlink-deployments-framework": minor
---

feat(evm): add built-in gas defaults for selected EVM chains

CLDF now applies chain-specific fixed gas limits and legacy gas prices when
loading EVM chains. Base and Optimism estimates receive a 25% gas buffer.
Overrides on EIP-7825 chains are capped at 16,777,216 gas.

Most chains remain unchanged. Consumers can remove equivalent
`UpdateBlockchainsWithEVMGasOverrides` calls for configured chains.
