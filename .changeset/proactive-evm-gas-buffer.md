---
"chainlink-deployments-framework": minor
---

feat(evm): add proactive gas limit buffer for on-chain transactions

CLDF now applies a configurable gas limit buffer to EVM transactions to reduce out-of-gas failures caused by optimistic `eth_estimateGas` results.

### Default behavior

- Production EVM chains loaded via `engine/cld/chains` use `evm.DefaultGasLimitBufferBps` (**2500 = +25%**).
- The buffer is applied in two places:
  1. **`MultiClient.EstimateGas`** — all auto-estimated txs (operations, operations2, MCMS, legacy CLI).
  2. **Explicit `GasLimit` overrides** on `operations2` `DeployInput` / `FunctionInput` — manual limits are also padded.
- You only pay for gas **used**, not the limit. The buffer is free headroom, outside of exceptions like Hedera.
- Test/sim providers (Anvil, CTF) are unchanged: buffer defaults to **0** unless configured.

### When to rely on the default buffer

Use the default (+25%) for chains where estimation is usually close but occasionally under-shoots.

No action needed if you load chains through CLD's chain loader.

### Per-chain configuration

Disable or tune the buffer when wiring a custom `RPCChainProvider`:

```go
evmclient.WithGasLimitBufferBps(0)    // disable
evmclient.WithGasLimitBufferBps(3000) // +30%
```

In `engine/cld/chains`, branch on chain selector to apply chain-specific opts. Example pattern for a chain with bad estimation:

```go
clientOpts := []func(*evmclient.MultiClient){ /* retry config */ }
if selector != inkSelector {
    clientOpts = append(clientOpts, evmclient.WithGasLimitBufferBps(fevm.DefaultGasLimitBufferBps))
}
```

### Explicit operation-level gas limits

`GasLimit` on `DeployInput` / `FunctionInput` bypasses estimation but **still receives the buffer** (10M → 12.5M at default settings). Use this when you want a padded manual limit, not an exact cap.

For an **exact** limit with no buffer, set it on the deployer key via `WithGasLimit` instead of on the operation input.

### API surface

| Symbol | Package | Purpose |
|--------|---------|---------|
| `evm.DefaultGasLimitBufferBps` | `chain/evm` | Default +25% constant |
| `evm.ApplyGasLimitBuffer` | `chain/evm` | Shared buffer math |
| `evm.GasLimitBufferBpsFromClient` | `chain/evm` | Read buffer from client |
| `rpcclient.WithGasLimitBufferBps` | `chain/evm/provider/rpcclient` | Configure per `MultiClient` |
| `provider.WithGasLimit` | `chain/evm/provider` | Fixed deployer gas limit (skips estimate) |

### Not covered

- ZkSync VM deploys (separate wallet path)
- Hardcoded 21k ETH transfer txs
- Simulated txs (`NoSend: true`)
