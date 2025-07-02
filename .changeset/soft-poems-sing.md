---
"chainlink-deployments-framework": minor
---

feat(evm): support signing hash

Introduce a new field on the Evm Chain struct `SignHash` which accepts a hash a signs it , returning the signature.

This feature has been requested by other teams so they dont have to use the `bind.TransactOpts` to perform signing.

FYi This has BREAKING CHANGE due to interface and field rename, i decided to not have alias because the usage is limited to CLD which i will update immediately. after this is merged.

Migration guide:
```
interface TransactorGenerator -> SignerGenerator
field ZkSyncRPCChainProviderConfig.SignerGenerator -> ZkSyncRPCChainProviderConfig.ZkSyncSignerGen
```
