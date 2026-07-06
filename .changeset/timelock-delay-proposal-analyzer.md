---
"chainlink-deployments-framework": minor
---

Add a built-in timelock delay validator to `analyze-proposal-v2` that compares schedule proposal delays against on-chain `minDelay`. Thread proposal `ChainMetadata` through `ExecutionContext` so chain-family timelock inspectors (e.g. Sui) can be built correctly. Extends the exported `ExecutionContext` interface with proposal metadata accessors.
