---
"chainlink-deployments-framework": patch
---

fix(OnlyLoadChainsFor)!: remove migration name parameter for environment option

BREAKING CHANGE: The `environment` option in `OnlyLoadChainsFor` no longer accepts a migration name parameter. The name parameter was only used for logging which is not necessary.

### Usage Migration

**Before:**

```go
environment.OnlyLoadChainsFor("analyze-proposal", chainSelectors), cldfenvironment.WithoutJD())
```

**After:**

```go
environment.OnlyLoadChainsFor(chainSelectors), cldfenvironment.WithoutJD())
```
