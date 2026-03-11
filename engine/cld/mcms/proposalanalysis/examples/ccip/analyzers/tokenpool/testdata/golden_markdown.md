## Proposal — ccip (mainnet)


<details>
<summary><h3>Batch 1 — ethereum-mainnet (<code>5009297550715157269</code>)</h3></summary>


#### Call 1

- [ ] **LockReleaseTokenPool 1.5.1** `applyChainUpdates`

**Target:** `0x1234567890abcdef1234567890abcdef12345678`

**Inputs:**

- **`remoteChainSelectorsToRemove`** (`uint64[]`): <nil>
- **`chainsToAdd`** (`tuple[]`): (decoded)


**Changes:**

- **outbound to bsc-mainnet capacity:** ~~1 USDC (1,000,000, decimals=6)~~ -> **2 USDC (2,000,000, decimals=6)**
- **outbound to bsc-mainnet rate:** ~~0.0001 USDC (100, decimals=6)~~ -> **0.0002 USDC (200, decimals=6)**
- **inbound from bsc-mainnet capacity:** ~~0~~ -> **0.5 USDC (500,000, decimals=6)**
- **inbound from bsc-mainnet rate:** ~~0~~ -> **0.00005 USDC (50, decimals=6)**


_Annotations:_
- ccip.token.symbol: USDC
- ccip.token.decimals: 6
- ccip.token.address: 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48
- ccip.chain_update: bsc-mainnet (11344663589394136015) added
- ccip.rate_limiter: inbound from bsc-mainnet: rate limiter enabled


#### Call 2

- [ ] **BurnMintTokenPool 1.5.1** `applyChainUpdates` ⚠ **warning** 🟡 risk: **medium**

**Target:** `0xabcdefabcdefabcdefabcdefabcdefabcdefabcd`

**Inputs:**

- **`remoteChainSelectorsToRemove`** (`uint64[]`): [ 3734025351759498498 ]
- **`chainsToAdd`** (`tuple[]`): <nil>


_Annotations:_
- ccip.chain_update: avalanche-mainnet (6433500567565415381) removed


</details>
