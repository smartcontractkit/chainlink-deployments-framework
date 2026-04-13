---
"chainlink-deployments-framework": minor
---

feat(operations-api): introduce `WithForceExecute` and new `ExecuteOperationN` options

### Migration: `ExecuteOperationN`

**Signature**

- Before: `ExecuteOperationN(..., opts ...ExecuteOption[IN, DEP])`
- After: `ExecuteOperationN(..., opts ...ExecuteOperationNOption[IN, DEP])`
