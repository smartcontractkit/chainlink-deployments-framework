# chainlink-deployments-framework

## 0.1.1

### Patch Changes

- [#81](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/81) [`ba1cd63`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ba1cd6338dd7e0efc087e579ffe6c2f6dd5d8b3f) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix incorrect error message order

- [#83](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/83) [`a3b78c6`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a3b78c6e1caa60a0dac056f5af2678fa44802831) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - enhance multiclient logging

## 0.1.0

### Minor Changes

- [#69](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/69) [`ee24199`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ee24199564bd04fa66dc24152b0cbc263d37f7ac) Thanks [@tt-cll](https://github.com/tt-cll)! - adds solana program close and support for deploying without extended buffers

### Patch Changes

- [#74](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/74) [`98904ba`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/98904ba591ae12bbeb9ef48b15c0bcc990e934c8) Thanks [@giogam](https://github.com/giogam)! - feat(multiclient): wraps debug calls with retry logic

## 0.0.14

### Patch Changes

- [#39](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/39) [`08e4660`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/08e46605ecb71276b7a35aa94887473da5ee08fb) Thanks [@jkongie](https://github.com/jkongie)! - BREAKING: Operations retry logic is now opt in. Use the `WithRetry` method in your `ExecuteOperation` call to enable retries

## 0.0.13

### Patch Changes

- [#64](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/64) [`f05efd9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f05efd9b0e417da9e6b0fd53372566584ad65520) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - migrate more helpers for writing changesets

## 0.0.12

### Patch Changes

- [#62](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/62) [`e31e3ea`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e31e3eab10cd8cbb10a3ee17ae4202fb7f9f495f) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - Add more helpers from chainlink/deployment which are useful for writing changesets

## 0.0.11

### Patch Changes

- [#59](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/59) [`5d5a317`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5d5a317363b549ac372b9f1c0430ff9566d4314d) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - better multiclient logging and expose dial attempts and delay as config

## 0.0.10

### Patch Changes

- [#51](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/51) [`4e85039`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4e85039f9692ff325f60a863801a1db674fad32e) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - set multiclient retry to 1

## 0.0.9

### Patch Changes

- [#49](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/49) [`937b6c9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/937b6c91874b5b5fe0749141a95071d2d4725026) Thanks [@giogam](https://github.com/giogam)! - feat(multiclient): wraps HeaderByNumber calls with retry logic

## 0.0.8

### Patch Changes

- [#43](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/43) [`1691ef8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1691ef8020546b400b0ec939616b8dea187f94cb) Thanks [@giogam](https://github.com/giogam)! - feat(multiclient): adds debug logs to ethclient direct calls

## 0.0.7

### Patch Changes

- [#44](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/44) [`f3f046f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f3f046f4fd3b0e24aac0abeb5208dafa7f8ba263) Thanks [@ajaskolski](https://github.com/ajaskolski)! - chore:updates for aptos from chainlink

## 0.0.6

### Patch Changes

- [#40](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/40) [`4331827`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/43318272cbc9a58723c6e54bbdaaa2c98ef6d3b2) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat: migrate chains and environment from chainlink repo
  feat: migrate changeset and changeset output from chainlink repo

## 0.0.5

### Patch Changes

- [#35](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/35) [`b9ae659`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b9ae659fd4e27010fa29fcc8dac11fb55f2efe24) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat: migrate ocr secrets

- [#36](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/36) [`455301e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/455301ea3e5672e70dd5887b1675f6ec29cf577a) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat:add OffchainClient interface

- [#31](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/31) [`416e1a4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/416e1a42c537591a3c8a7f37cfcf6f54aeb9ee1a) Thanks [@jkongie](https://github.com/jkongie)! - Datastore labels are now serialized into JSON as an array of strings

- [#33](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/33) [`520d47b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/520d47bf82a9500bb7109e22559698ea6431f548) Thanks [@jkongie](https://github.com/jkongie)! - Adds an `operations/optest` package containing test utility functions.

- [#30](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/30) [`2d3c56c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2d3c56cfbc4878ad8b705915def695a2fffc9d7b) Thanks [@akuzni2](https://github.com/akuzni2)! - Add an address filter to datastore

## 0.0.4

### Patch Changes

- [#23](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/23) [`da5bdfa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/da5bdfa13fbaee4f51900b5a92ea5b450968fff4) Thanks [@ajaskolski](https://github.com/ajaskolski)! - Add datastore

- [#25](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/25) [`6de2361`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6de236197630f604950c683af601f776f3e44444) Thanks [@giogam](https://github.com/giogam)! - feat: adds datastore conversion utilities

## 0.0.3

### Patch Changes

- [#14](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/14) [`ce85835`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ce858356ce2da7cd2a5ccc607f8569d2641096e5) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - downgrade go-etherum to align with chainlink repo

## 0.0.2

### Patch Changes

- [`31f0a6e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/31f0a6ec6ad3289c3bd84e9f4f8765033a5b94cd) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - First release Test
