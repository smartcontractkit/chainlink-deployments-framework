---
"chainlink-deployments-framework": patch
---

Bump chainlink-testing-framework/framework to pick up the Sui/Aptos CTF provider cwd-tar-race fix (upstream smartcontractkit/chainlink-testing-framework#2519). Resolves intermittent `archive/tar: write too long` flakes in `Test_CTFChainProvider_Initialize` for the Sui and Aptos providers.
