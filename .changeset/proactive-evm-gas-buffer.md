---
"chainlink-deployments-framework": minor
---

feat(evm): add built-in gas defaults for selected EVM chains

CLDF applies chain-specific gas configuration at load time via built-in defaults in `engine/cld/chains`. No `networks.yaml` metadata is required.

### Default behavior

- **No gas overrides by default** — most EVM chains use normal estimation with no buffer.
- **Base and Optimism mainnet and Sepolia testnets** get a **+25%** gas limit buffer on `eth_estimateGas` results.
- Chains with built-in overrides that enforce **EIP-7825** (Base, Optimism, Metal, BOB, Ink Sepolia, Zora testnet) also cap gas at **16,777,216** on estimates and fixed deployer limits.
- **Fixed deployer gas overrides** replace consumer-side `UpdateBlockchainsWithEVMGasOverrides` for:
  - Metal, Hedera, BOB, Wemix, MegaETH, Edge, Bittensor, Mind, Ronin mainnet **and testnet** (fixed gas limit and/or legacy gas price).
  - Testnet-only: Gnosis Chiado, Ink Sepolia, Zora, Ronin Saigon, Ethereum Sepolia Ronin.

### Migration from `UpdateBlockchainsWithEVMGasOverrides`

Remove consumer-side gas override calls for the chains listed above. Base and Optimism chains get +25% buffer automatically.

### API surface

| Symbol | Package | Purpose |
|--------|---------|---------|
| `chains.BaseGasLimitBufferBps` | `engine/cld/chains` | +25% estimate buffer for Base and Optimism chains |
| `chains.HederaDeployerGasPriceWei` | `engine/cld/chains` | Fixed Hedera legacy gas price (1500 gwei) |
| `evm.EIP7825MaxTxGasLimit` | `chain/evm` | EIP-7825 per-transaction gas cap (16,777,216) |
| `evm.ApplyGasLimitWithBufferAndCap` | `chain/evm` | Apply bps buffer then cap gas limit |
| `evm.MaxTxGasLimitFromClient` | `chain/evm` | Read max tx gas cap from RPC client |
| `provider.WrapSignerWithGasOverrides` | `chain/evm/provider` | Fixed deployer gas limit/price |
