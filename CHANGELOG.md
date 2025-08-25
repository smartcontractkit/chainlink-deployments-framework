# chainlink-deployments-framework

## 0.34.0

### Minor Changes

- [#316](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/316) [`7c113b3`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7c113b3a30f017fd8541d45d38cfbf93f6120405) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Move config resolver framework into CLDF

### Patch Changes

- [#318](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/318) [`fb41871`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fb41871d33d5ba8c0df1cc7f1bca2dee8137684c) Thanks [@giogam](https://github.com/giogam)! - feat: adds ci config files path methods

- [#278](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/278) [`51ef269`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/51ef2692b7108894649a8ead612e502724e5b80f) Thanks [@cgruber](https://github.com/cgruber)! - Adds an in-memory catalog implementation for testing.

## 0.33.0

### Minor Changes

- [#312](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/312) [`4ef3084`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4ef30847da776685b9bd38c67f9487bd10182a83) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] Fixes domain config path getters to have more consistent naming

### Patch Changes

- [#313](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/313) [`0f3e368`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0f3e3687e00101917787e6893ca549dd268ec0a5) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(anvil): fix nil pointer when T is not provided

## 0.32.0

### Minor Changes

- [#308](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/308) [`11daf8d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/11daf8de3cb89d01525988fcc427a45cf56ca29f) Thanks [@jkongie](https://github.com/jkongie)! - Adds additional config related file path getter methods to Domains

- [#311](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/311) [`ae71e08`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ae71e085d407cd0102a6c5a571678bdeb77cf5bf) Thanks [@ecPablo](https://github.com/ecPablo)! - Add support for gas limit option on raw signer generator

## 0.31.0

### Minor Changes

- [#303](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/303) [`c22683e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c22683ebe97aba5755cef1492220a3c2b05cec2a) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - Add Sui chain

- [#306](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/306) [`f876eea`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f876eeac15b6e4ac6921ba96d2323f346cd22a5d) Thanks [@faisal-chainlink](https://github.com/faisal-chainlink)! - Added optional configs for Sui CTF provider config to allow specifying image and platform.

## 0.30.0

### Minor Changes

- [#297](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/297) [`c092120`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c0921203f73f596d024cdcb2d4bc180056688652) Thanks [@jkongie](https://github.com/jkongie)! - Allow users to marshal and unmarshal env config directly

- [#298](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/298) [`c754f68`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c754f68e56e44e95dd0a46deb6c5d864e2c194f5) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: New EVM CTF Anvil Provider

### Patch Changes

- [#300](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/300) [`4ff7f93`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4ff7f9323e82524f73385ad41cdc3f1e5220e938) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - Moved analyze.go from chainlink repo

- [#296](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/296) [`b4ba277`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b4ba2773290fdbc5be5c4210ab184bdab7258132) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - Fix message encoding in Sui signing

## 0.29.0

### Minor Changes

- [#294](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/294) [`6e35a51`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6e35a51007ab51169cf258c532cbfadd7caf83ab) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor: extract chains network and minimal env dependency from cld

## 0.28.0

### Minor Changes

- [#288](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/288) [`762cddd`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/762cddd7ce357c5a5d37154a51170c91fec83686) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - Add Sui env config support

- [#269](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/269) [`b25a886`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b25a8865f7d6d59c7821a6d8bb10372fc9941781) Thanks [@vreff](https://github.com/vreff)! - #changed bump chainlink-common, update Keystore to implement extended Keystore interface

- [#289](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/289) [`6a5acfb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6a5acfbb6e01999aafcc9cf9f3621bb803e3f7e0) Thanks [@jkongie](https://github.com/jkongie)! - Network configuration is now validated on load. It ensures that the type and chain selector are present, as well as at least one RPC.

- [#283](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/283) [`f89cfad`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f89cfada6ea8de2b91f5a17cd5881d1efdd71079) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] JobDistributorConfig.Auth is now a pointer to indicate that it is an optional field

### Patch Changes

- [#291](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/291) [`af3df24`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/af3df247572945d736a0b098cbb34703eab8ea24) Thanks [@jkongie](https://github.com/jkongie)! - Fixes legacy env vars fallback for certain fields on the env.Config

## 0.27.0

### Minor Changes

- [#276](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/276) [`55e476b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/55e476ba237aaf2aadf3e0327a649a3b0ce925e2) Thanks [@jkongie](https://github.com/jkongie)! - Bump the mcms library to v0.21.1

### Patch Changes

- [#281](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/281) [`9c26dea`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/9c26deaf567f67f14eaae88cbc58f76fb6d180a2) Thanks [@ajaskolski](https://github.com/ajaskolski)! - chore(domain): export SetupTestDomainsFS and rootPath for easier granular refactor

## 0.26.0

### Minor Changes

- [#279](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/279) [`7ffac78`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7ffac78a9ce16170c77db3bc87e9bb9a311c8cfa) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor(engine): domain move view state from cld to cldf

## 0.25.0

### Minor Changes

- [#272](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/272) [`10bd095`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/10bd0955f1b81033d39a6a338b8337f63c2dab1a) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - Introdce KMS signer for TRON

- [#277](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/277) [`372d4c5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/372d4c51481f5e16420efa36ff5aeaa9f8f69481) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(tron): signer generator no longer lazy loads

### Patch Changes

- [#245](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/245) [`5215932`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/52159327fe87bd88cb23e17fd422fd1b5ab76c01) Thanks [@cgruber](https://github.com/cgruber)! - Implement transactional logic in Catalog backed datastore APIs.

## 0.24.0

### Minor Changes

- [#270](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/270) [`38e003a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/38e003a9baf16313515bd8344e729e7a220b5a7b) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] Removes `WithRPCURLTransformer` load option with 2 separated options targeting HTTP and WS separately (`WithHTTPURLTransformer` and `WithWSURLTransformer`).

- [#261](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/261) [`023116b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/023116b9b4eb7f9e0e645f67b72af6ce159d217c) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor: moved domain from cld pkg/migrations to cldf

## 0.23.0

### Minor Changes

- [#257](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/257) [`f051994`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f0519945aca878c4a6728a26c0c818aea5498e5b) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - Add Sui chain providers for RPC and CTF

- [#264](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/264) [`b9ef148`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b9ef148b8e2b29d5651b6552ef2bd60120dd0aad) Thanks [@eduard-cl](https://github.com/eduard-cl)! - Update chainlink-evm gethwrappers version

## 0.22.0

### Minor Changes

- [#249](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/249) [`01b951b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/01b951bc5bf9c1ba8edd1a620819a2c023409edd) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor: migrates nodes management logic from cld

- [#259](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/259) [`1d96752`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1d9675226efc16bff68b23772e70ceb6da962582) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] The Load function of Network Config has been changed to simplify the URL transformation option

- [#258](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/258) [`0e7b13c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0e7b13c5f817d2da1a019ebfd95520f215a10e1c) Thanks [@jkongie](https://github.com/jkongie)! - Adds configuration loading to the CLD engine

- [#251](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/251) [`6c1338e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6c1338e66edfe0cccaeae91aaa1a1dd9074999da) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor: adds files and json utils from cld

### Patch Changes

- [#252](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/252) [`4d57885`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4d57885f3b3248a499f1a22744f8418144000236) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - BREAKING: remove deployment.OffchainClient. Use offchain.Client instead

  Migration Guide:

  ```
  cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment" -> cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
  cldf.OffchainClient -> offchain.Client
  ```

- [#256](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/256) [`afca1be`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/afca1be4cfc6a7e32694a069289ef27f00105e0a) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - use dev tagged image for TON CTF Provider

- [#253](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/253) [`f8876aa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f8876aa46bfdde3f06c8f98d133d27f5320cfd14) Thanks [@eduard-cl](https://github.com/eduard-cl)! - Refactor the Tron package options to be pointers in order to support optional configuration.

## 0.21.0

### Minor Changes

- [#246](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/246) [`cf4cb13`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/cf4cb131a1ec9cf021e3ca29e4422f641bc23e2b) Thanks [@jkongie](https://github.com/jkongie)! - Adds a network configuration package to the CLD engine

- [#241](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/241) [`440d5ea`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/440d5ea70c6138075aa908309512cd062b8c52ce) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: new CTF provider for JD

### Patch Changes

- [#243](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/243) [`e45f09f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e45f09fdb65a1dd355355a757a7af6e0ded6eec2) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - remove gogo/protobuf replace use v1.3.2

## 0.20.0

### Minor Changes

- [#236](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/236) [`01e7343`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/01e73431465cd84c334ae234b388a0b918dd2854) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: new JD client and offchain provider for JD

## 0.19.0

### Minor Changes

- [#219](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/219) [`c797129`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c797129296135df9cab9a991d93db8e0c44ae02c) Thanks [@eduard-cl](https://github.com/eduard-cl)! - feat: introduce tron chain provider and ctf provider

## 0.18.0

### Minor Changes

- [#230](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/230) [`149c03f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/149c03f6028011d5dd7cd40f739fa89f32c4462d) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - introduce Catalog GRPC client

### Patch Changes

- [#229](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/229) [`4ea8e79`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4ea8e790a12017348783df9079c3c86fd72664fd) Thanks [@giogam](https://github.com/giogam)! - Catalog Datastore #2

## 0.17.3

### Patch Changes

- [#222](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/222) [`fe4e84e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fe4e84ede1ca415f9ff277c3996da8f5d97b58c8) Thanks [@stackman27](https://github.com/stackman27)! - update sui sdk

## 0.17.2

### Patch Changes

- [#220](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/220) [`1d13a2d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1d13a2d622460265c65be012e3a5946df650fd9f) Thanks [@tt-cll](https://github.com/tt-cll)! - updates solana close

## 0.17.1

### Patch Changes

- [#215](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/215) [`d748632`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d748632cbb80494ca17dbbdf62a3252c0e08d418) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: improve log message on multiclient healthcheck

## 0.17.0

### Minor Changes

- [#212](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/212) [`c4279c9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c4279c96bc1e14b3ec7e2093de90b58f5a7ecd27) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(ton): new ctf provider

## 0.16.0

### Minor Changes

- [#208](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/208) [`a3fd06a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a3fd06aa3e0d36d20dac3b2888cc93a4897ee372) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(evm): support signing hash

  Introduce a new field on the Evm Chain struct `SignHash` which accepts a hash a signs it , returning the signature.

  This feature has been requested by other teams so they dont have to use the `bind.TransactOpts` to perform signing.

  FYi This has BREAKING CHANGE due to interface and field rename, i decided to not have alias because the usage is limited to CLD which i will update immediately. after this is merged.

  Migration guide:

  ```
  interface TransactorGenerator -> SignerGenerator
  field ZkSyncRPCChainProviderConfig.SignerGenerator -> ZkSyncRPCChainProviderConfig.ZkSyncSignerGen
  ```

## 0.15.1

### Patch Changes

- [#202](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/202) [`61bee71`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/61bee71ccebcdedd3097005cccbc2c1c6bd413c9) Thanks [@jkongie](https://github.com/jkongie)! - Additional users generated by the Simulated EVM Chain Provider are now prefunded

- [#203](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/203) [`60d5bd2`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/60d5bd22518eef7ffd6014b56197bba0c20692c9) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: handle empty data on parseErrorFromABI

## 0.15.0

### Minor Changes

- [#198](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/198) [`68696ef`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/68696ef5ac683418c7da7a3e1693fc09dbe0537f) Thanks [@jkongie](https://github.com/jkongie)! - Adds a `Backend` method to the `SimClient` to return the underlying simulated backend

- [#201](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/201) [`b57839d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b57839d68b0109997e1e42a1b6abb941337b833f) Thanks [@jkongie](https://github.com/jkongie)! - Adds a new ZkSync Chain Provider

## 0.14.0

### Minor Changes

- [#184](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/184) [`e39a622`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e39a622603a2e84e0789c2c6441533ccd89fee5c) Thanks [@jkongie](https://github.com/jkongie)! - Adds a Simulated EVM Chain Provider using the go-ethereum `simulated` package as the backend.

- [#186](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/186) [`8c6b0eb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8c6b0eb6c58ddb6361a554bfd634bf4e312b6250) Thanks [@jkongie](https://github.com/jkongie)! - Adds an EVM RPC Chain Provider

### Patch Changes

- [#192](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/192) [`5375acd`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5375acd99f52398a01de97107154c1c76860adda) Thanks [@jkongie](https://github.com/jkongie)! - The ZkSync Chain Provider will retry up to 10 times to boot the container

- [#197](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/197) [`bf0aa29`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bf0aa294ddd0e2d419207039c57476c15e5b0a83) Thanks [@jkongie](https://github.com/jkongie)! - The websocket url in the Solana Chain Provider is now optional

- [#187](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/187) [`3d8c945`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/3d8c945d92915b28f771662a49fe38db42d3d1ba) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(operations): optimization on dynamic execution

## 0.13.1

### Patch Changes

- [#177](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/177) [`bf62b62`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bf62b62cb4b8afdafc234f805888b94cf503e293) Thanks [@giogam](https://github.com/giogam)! - feat(datastore): adds ChainMetadata

- [#180](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/180) [`2da17fa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2da17fa445fc2f00d590ccbd977aef127717dec9) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(operations): fix nil OperationRegistry on ExecuteSequence

## 0.13.0

### Minor Changes

- [#174](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/174) [`4a4f9b2`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4a4f9b2d9e0ff5e45f89e609379d470c28f0bc93) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: support dynamic execution of operation

- [#166](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/166) [`f5a2ca9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f5a2ca9248e22b653a723f75b2a55f7e37675312) Thanks [@jkongie](https://github.com/jkongie)! - Adds a zkSync CTF provider under the EVM Chain

## 0.12.1

### Patch Changes

- [#172](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/172) [`d162d8a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d162d8a2d38c19d1f63683b7ed7631f895f01a01) Thanks [@jkongie](https://github.com/jkongie)! - Change ccip-solana version to match chainlink/deployment

## 0.12.0

### Minor Changes

- [#165](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/165) [`5df5ef6`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5df5ef63dfbbf926c0507743c196a58c267320c4) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] The `chain.Provider` `Initialize` method now requires a context to be provided.

- [#152](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/152) [`662acb2`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/662acb2f43991d56f6e554968904420cc7b7ef21) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(operations): introduce ExecutionOperationN

### Patch Changes

- [#162](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/162) [`af44a35`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/af44a357d6976071aba41a52d438a5f8faed1ed5) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Remove address book types that arent used

- [#161](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/161) [`fb0c82e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fb0c82e6ddc54d3466dfe80cbb331319936e9cbd) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Revert PR 156 as not needed in CLDF

- [#154](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/154) [`88373b1`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/88373b1beeefb01e350327abd6f8a607fe556a54) Thanks [@jkongie](https://github.com/jkongie)! - Solana Chain now provides a SendAndConfirm method which is intended to replace the Confirm method.

- [#163](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/163) [`bbf7434`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bbf743441b8da517369f9dc3d622f87341ae37e6) Thanks [@jkongie](https://github.com/jkongie)! - Update chainlink-testing-framework/framework packages to 0.9.0 to fix a flakey test

- [#164](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/164) [`8019439`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/80194395e077d1d6255869dc00771b8cbe3e92d3) Thanks [@jkongie](https://github.com/jkongie)! - Adds a Solana CTF Chain Provider which returns a chain backend by a testing container

## 0.10.0

### Minor Changes

- [#156](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/156) [`e92e849`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e92e8498d3276d210cbf3e4383deb99e26d82718) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Added custom JSON marshaling methods to AddressBookMap and AddressesByChain types to ensure deterministic JSON output with chain selectors ordered numerically and addresses ordered alphabetically (case-insensitive).

## 0.9.1

### Patch Changes

- [#149](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/149) [`7af7a51`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7af7a51d8f414e49bcb4cfd8937225ff24e586cf) Thanks [@giogam](https://github.com/giogam)! - feat(datastore) remove generics from the top level api

- [#150](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/150) [`1134c51`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1134c51f19c79d8c51ca8bc5fb165b905b416167) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: ChildOperationReports not set correctly

## 0.9.0

### Minor Changes

- [#132](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/132) [`f2929a9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f2929a9489f00b67e9c0825bf80af447b505b1bd) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - feat: reorder bad RPCs if they fail all retries

### Patch Changes

- [#148](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/148) [`105798d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/105798d85db65ac8d96fb20247d15e5f10ff22d2) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - remove NewCLDFEnvironment in favour of NewEnvironment

- [#141](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/141) [`c80685e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c80685e2099cd04ae43e89292e52294d736bfff7) Thanks [@giogam](https://github.com/giogam)! - feat(datastore): removes Clone requirement for custom metadata types

## 0.8.2

### Patch Changes

- [#146](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/146) [`e7804b8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e7804b878e63441b9fe9559366fcb2d438877bc1) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - refactor: use chain sel from loop instead of chain.ChainSelector()

- [#137](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/137) [`925f216`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/925f216f2b4dccbc8e0200d9beadb393338ece3d) Thanks [@jkongie](https://github.com/jkongie)! - Fixes Aptos Chain Providers return a Chain pointer instead of value

## 0.8.1

### Patch Changes

- [#133](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/133) [`06ecd68`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/06ecd6875ebc08a0e4edb0171ae41e914ec335b9) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - remove legacy chains field

- [#138](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/138) [`41ec05c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/41ec05cea2718fb5fc6d34706e45e183db336220) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - support returning BlockChain that are pointers

## 0.8.0

### Minor Changes

- [#134](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/134) [`a09cfcb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a09cfcbfc07d144d3f96fe728c8ebbc8e0be9277) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - New blockchain.AllChains() and NewBlockChainFromSlice constructor

### Patch Changes

- [#136](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/136) [`ae801d5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ae801d51846f59147d5f69295cdcaf549467ea04) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - rename AllChains to All

## 0.7.0

### Minor Changes

- [#128](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/128) [`a4fe4bf`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a4fe4bfe51a946a9a9d9840908510cca550e512f) Thanks [@jkongie](https://github.com/jkongie)! - Remove unused `deployment.AllDeployerKeys` function

### Patch Changes

- [#123](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/123) [`aa4c5e6`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/aa4c5e6ed87b70349d8a5763429c7ec12329853b) Thanks [@jkongie](https://github.com/jkongie)! - Adds Aptos Chain providers for RPC and CTF backed chains

## 0.6.0

### Minor Changes

- [#125](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/125) [`870e061`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/870e061809d9e20acd6ce13022d2150f59d55df4) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: expose error and attempts on the inputHook

## 0.5.1

### Patch Changes

- [#114](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/114) [`20b09f9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/20b09f90628d03171e03c2caa9fd47bccb94b867) Thanks [@jkongie](https://github.com/jkongie)! - Add `Exists` and `ExistsN` methods to `Blockchains` to test for the existence of a chain for the provided selector/s

- [#117](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/117) [`92c030d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/92c030db30a01c602f21b09b2b6a6766dea4065c) Thanks [@giogam](https://github.com/giogam)! - feat(datastore): Implement Stringer for ContractMetadata and AddressRef Keys

- [#115](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/115) [`3f32425`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/3f32425197babacd1baf406756cfb458276651b6) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: remove unused error return value

- [#119](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/119) [`bb450f4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bb450f4de6fe286f942d5e87a0e7a05af357c7ef) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - ListChainSelectors can filter multiple families

- [#112](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/112) [`91ac227`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/91ac2274033650e3cfbfe72815e639e61e3e0229) Thanks [@jkongie](https://github.com/jkongie)! - Adds a `Family` method to the `Blockchain` interface

- [#116](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/116) [`8eaef28`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8eaef2828ad73614868927af32e4a67666014aee) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - chains: update sui & ton to compose ChainMetadata

- [#118](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/118) [`afc5f2f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/afc5f2f300a8588af2061b1dea238d2774d2212c) Thanks [@giogam](https://github.com/giogam)! - feat(datastore): removes EnvMetadataKey implementation

## 0.5.0

### Minor Changes

- [#108](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/108) [`c1e68b7`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c1e68b7ddbe869671d9c61b4d1d45e77d801a1c2) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: new blockchains field in environment

- [#98](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/98) [`bdddc37`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bdddc37b7cb7c01833c8019e9baff8dc88f664a8) Thanks [@cfal](https://github.com/cfal)! - add Sui chain support

- [#110](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/110) [`e52fe33`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e52fe33bd77f1eb67e384d4dccc9bd12d0b9d1b8) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: add ton chain

## 0.4.0

### Minor Changes

- [#100](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/100) [`a975e70`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a975e70cc0c4652543e1b8fa5c68aa8d691c32ef) Thanks [@jkongie](https://github.com/jkongie)! - The deprecated `Proposals` field on the `ChangesetOutput` has been removed in favour of `MCMSTimelockProposals`

## 0.3.0

### Minor Changes

- [#95](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/95) [`6ea8603`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6ea86035fc3effc6da0d7047c59e940f1340061b) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: Concurrency support for Operations API

## 0.2.0

### Minor Changes

- [#88](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/88) [`18c3cf2`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/18c3cf210bcd1cd780d0f707337da8aed47bbc23) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Adding RPC client health check after successful dial

## 0.1.3

### Patch Changes

- [#84](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/84) [`b3fec25`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b3fec25041be39e035eb7fcbfb3139948bb621d9) Thanks [@giogam](https://github.com/giogam)! - feat(multiclient): adds timeouts to retryWithBackups and dialWithRetry

## 0.1.2

### Patch Changes

- [#85](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/85) [`4285f35`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4285f359dbd872203590661e27512f9d2672a7bd) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(chain): refactor solana DeployProgram to accept ProgramBytes

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
