---
"chainlink-deployments-framework": minor
---

feat: add Canton chain support for CLD engine and chainlink-deployments

Add Canton as a supported chain so it can be loaded from the chainlink-deployments repo via the CLD engine.

- **Config**: `CantonConfig` with `JWTToken`, env binding `ONCHAIN_CANTON_JWT_TOKEN`, and `CantonMetadata` for network participant config
- **Chain loader**: Canton RPC chain loader in `engine/cld/chains`; loads from network metadata + JWT secret
- **MCMS adapter**: `CantonChains()` on `ChainsFetcher` and `CantonChain(selector)` on `ChainAccessAdapter`
- **Tests**: Config env tests and optional YAML test data updated for Canton; MCMS adapter tests for Canton chain access

Existing RPC and CTF Canton providers in `chain/canton/provider` are unchanged; they are now wired into the engine and MCMS adapter.
