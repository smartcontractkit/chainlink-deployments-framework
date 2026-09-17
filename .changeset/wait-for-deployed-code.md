---
"chainlink-deployments-framework": patch
---

Wait for deployed contract code to be visible via `eth_getCode` before returning from `NewDeploy`, to avoid downstream reads racing a lagging RPC backend right after deployment. Tests mocking `evm.Chain.Client` around a deploy now need to stub `CodeAt`.
