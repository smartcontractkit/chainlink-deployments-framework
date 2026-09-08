---
"chainlink-deployments-framework": patch
---

fix: confirm EVM deploys based on the returned tx instead of the chain type

`DeployContract` skipped confirmation whenever `chain.IsZkSyncVM` was set, which also
skipped it for EVM-emulator deploys on zkSync chains. Those return a real transaction, so a
dropped tx was recorded in the address book as if it had landed. Confirmation now keys off
`ContractDeploy.Tx`, which is nil only for native zkSync deploys.
