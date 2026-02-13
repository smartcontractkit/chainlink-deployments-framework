---
"chainlink-deployments-framework": minor
---

feat(chain): introduce lazy chain loading

Feature toggle under CLD_LAZY_BLOCKCHAINS environment variable to enable lazy loading of chains.
Migration guide:

- Previously: env.BlockChains.EVMChains()
- Preferred: env.Chains().EVMChains()

By using the newer Chains() method, you can now access newer features such as loading the chains lazily, which is useful for large environments.
