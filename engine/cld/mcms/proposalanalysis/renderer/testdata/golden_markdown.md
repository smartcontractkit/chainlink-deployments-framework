## Proposal — ccip (mainnet)


<details>
<summary><h3>Batch 1 — ethereum-mainnet (<code>5009297550715157269</code>)</h3></summary>



_Annotations:_
- batch.note: first batch

#### Call 1

- [ ] **OnRamp v1.5.0** `setRateLimiterConfig` ⚠ **warning** 🔴 risk: **high**

**Target:** `0x1111111111111111111111111111111111111111`

**Inputs:**

- **`target`** (`address`): 0xAbCdEf1234567890abcdef1234567890abcdef12
  - _label: destination contract_
- **`amount`** (`uint256`): 1,000,000,000,000,000,000
- **`enabled`** (`bool`): true
- **`proof`** (`bytes`): 0xdeadbeef
- **`destChainConfigArgs`** (`((uint64,(bool,uint16,uint32,uint32,uint32,uint8,uint8,uint16,uint32,uint16,uint16,bytes4,bool,uint16,uint32,uint32,uint64,uint32,uint32))[])`):
  ```text
  [
    {
      "DestChainSelector": "aptos-testnet (743186221051783445)",
      "DestChainConfig": {
        "IsEnabled": true,
        "MaxNumberOfTokensPerMsg": 1,
        "MaxDataBytes": 30000,
        "DestGasPerPayloadByteBase": 0,
        "DestDataAvailabilityMultiplierBps": 0,
        "ChainFamilySelector": "0xac77ffec",
        "GasMultiplierWeiPerEth": 1100000000000000000
      }
    },
    {
      "DestChainSelector": "sui-testnet (9762610643973837292)",
      "DestChainConfig": {
        "IsEnabled": true,
        "MaxNumberOfTokensPerMsg": 1,
        "MaxDataBytes": 16000,
        "DestGasPerPayloadByteBase": 16,
        "DestDataAvailabilityMultiplierBps": 1,
        "ChainFamilySelector": "0xc4e05953",
        "GasMultiplierWeiPerEth": 1100000000000000000
      }
    }
  ]
  ```
  - _note: multi-chain destination configuration_


**Changes:**

- **outboundRateLimit:** ~~0~~ -> **1,000,000**


_Annotations:_
- ccip.lane: ethereum -> arbitrum


#### Call 2

- [ ] **ERC20** `transfer`

**Target:** `0x2222222222222222222222222222222222222222`

**Inputs:**

- **`to`** (`address`): 0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef
- **`value`** (`uint256`): 500


</details>

<details>
<summary><h3>Batch 2 — binance_smart_chain-testnet (<code>13264668187771770619</code>)</h3></summary>


#### Call 1

- [ ] `pause`

**Target:** `0x3333333333333333333333333333333333333333`

</details>
