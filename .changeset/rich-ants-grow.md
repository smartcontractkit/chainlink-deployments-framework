---
"chainlink-deployments-framework": minor
---

Adds methods to determine the network type of a Chain

- `chain.NetworkType()` - Returns the network type determined by delegating to the `chain-selectors` package
- `chain.IsNetworkType(chainsel.NetworkTypeMainnet)` - Returns a boolean if the network type matches
