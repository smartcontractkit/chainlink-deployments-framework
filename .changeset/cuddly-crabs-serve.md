---
"chainlink-deployments-framework": patch
---

BREAKING: remove deployment.OffchainClient. Use offchain.Client instead

Migration Guide:

```
cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment" -> cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
cldf.OffchainClient -> offchain.Client
```
