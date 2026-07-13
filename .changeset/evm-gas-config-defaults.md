---
"chainlink-deployments-framework": minor
---

feat(evm): per-chain gas defaults via network YAML

Adds `gas_config` to EVM network metadata so domains can set default gas limit and price on the chain deployer at load time. Replaces hardcoded framework chain tables and consumer-side gas override workarounds.
