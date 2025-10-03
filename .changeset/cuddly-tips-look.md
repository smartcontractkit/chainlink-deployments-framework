---
"chainlink-deployments-framework": patch
---

Fixes test engine MCMS execution when multiple proposals have the same `validUntil` timestamp.

A salt override is added to each timelock proposal persisted to the state to ensure unique operation
IDs in test environments where multiple proposals may have identical timestamps. This salt is used
in the hashing algorithm to determine the root of the merkle tree.
