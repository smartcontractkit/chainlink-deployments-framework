---
"chainlink-deployments-framework": minor
---

feat(evm): add built-in gas defaults for selected EVM chains

CLDF applies chain-specific gas configuration at load time via built-in defaults in `engine/cld/chains`. No `networks.yaml` metadata is required.

### Default behavior

- **Default buffer is disabled** (`DefaultGasLimitBufferBps = 0`) for all EVM chains.
- **Base mainnet and Base Sepolia testnet** automatically get **+25%** gas buffer (`BaseGasLimitBufferBps = 2500`).
- Built-in deployer gas overrides replace consumer-side `UpdateBlockchainsWithEVMGasOverrides` for:
  - Metal, Hedera, BOB, Wemix, MegaETH, Edge, Bittensor, Mind, Ronin mainnet **and testnet** (fixed gas limit and/or legacy gas price).
  - Testnet-only: Gnosis Chiado, Ink Sepolia, Zora, Ronin Saigon, Ethereum Sepolia Ronin.

### Migration from `UpdateBlockchainsWithEVMGasOverrides`

Remove consumer-side gas override calls for the chains listed above. Base chains get +25% buffer automatically.

### API surface

| Symbol | Package | Purpose |
|--------|---------|---------|
| `evm.DefaultGasLimitBufferBps` | `chain/evm` | Default 0 (disabled) |
| `evm.BaseGasLimitBufferBps` | `chain/evm` | +25% for Base chains |
| `provider.WrapSignerWithGasOverrides` | `chain/evm/provider` | Fixed deployer gas limit/price |
