## Proposal — ccip (mainnet)


<details>
<summary><h3>Batch 1 — Chain <code>5009297550715157269</code></h3></summary>


_Annotations:_
- batch.note: first batch

#### Call 1

- [ ] **OnRamp v1.5.0** `setRateLimiterConfig` ⚠ **warning** 🔴 risk: **high**

**Target:** `0x1111..1111`

**Inputs:**

- **`target`** (`address`): 0xabcdef1234567890abcdef1234567890abcdef12
  - _label: destination contract_
- **`amount`** (`uint256`): 1,000,000,000,000,000,000
- **`enabled`** (`bool`): true

_Annotations:_
- ccip.lane: ethereum -> arbitrum


#### Call 2

- [ ] **ERC20** `transfer`

**Target:** `0x2222..2222`

**Inputs:**

- **`to`** (`address`): 0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
- **`value`** (`uint256`): 500


</details>

<details>
<summary><h3>Batch 2 — Chain <code>13264668187771770619</code></h3></summary>


#### Call 1

- [ ] `pause`

**Target:** `0x3333..3333`

</details>
