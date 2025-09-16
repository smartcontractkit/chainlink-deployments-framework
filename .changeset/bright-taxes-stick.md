---
"chainlink-deployments-framework": minor
---

**[BREAKING]** Refactored `LoadOffchainClient` to use functional options

## Function Signature Changed

**Before:**
```go
func LoadOffchainClient(ctx, domain, env, config, logger, useRealBackends)
```

**After:**
```go
func LoadOffchainClient(ctx, domain, cfg, ...opts)
```

## Migration Required

- `logger` → `WithLogger(logger)` option (optional, has default)
- `useRealBackends` → `WithDryRun(!useRealBackends)` ⚠️ **inverted logic**
- `env` → `WithCredentials(creds)` option (optional, defaults to TLS)
- `config` → `config.Offchain.JobDistributor`

**Example:**
```go
// Old
LoadOffchainClient(ctx, domain, "testnet", config, logger, false)

// New
LoadOffchainClient(ctx, domain, config.Offchain.JobDistributor,
    WithLogger(logger),
    WithDryRun(true), // Note: inverted!
)
