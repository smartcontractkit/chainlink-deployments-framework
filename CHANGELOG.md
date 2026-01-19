# chainlink-deployments-framework

## 0.75.1

### Patch Changes

- [#675](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/675) [`595b463`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/595b463ebb1834a97f99cf2c7559f0a0d0f09f28) Thanks [@ajaskolski](https://github.com/ajaskolski)! - fix(migration):include configuration check for datastore type all

## 0.75.0

### Minor Changes

- [#670](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/670) [`0a320ef`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0a320efc8afe4d52f6569b46ac81aa95ba7a14e4) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(engine): allow configuration for SUI chain in test engine

### Patch Changes

- [#668](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/668) [`ff9c85a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ff9c85a1c09ec9ba78e3073758e64502381e7a58) Thanks [@jkongie](https://github.com/jkongie)! - Bump `go-ethereum` to v1.16.8

## 0.74.3

### Patch Changes

- [#660](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/660) [`a8928d5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a8928d583390d89d496eb20269ca949bb55a59db) Thanks [@ecPablo](https://github.com/ecPablo)! - fix: avoid loading proposal ctx if the provider is nil
  chore: add deprecation warning to mcmsv2 commands

- [#666](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/666) [`d5bcb7c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d5bcb7cdf936230ddfa5f1cbff7774b3e4864ea5) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(jd): remove wsrpc from error message

- [#655](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/655) [`3791c84`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/3791c84cfd90e75e3b60261750e982ea5ac1a22d) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - feat: log from, to and raw data in forktests

- [#658](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/658) [`504cfaa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/504cfaa183399c6d86ee4b36d71239518322c8f3) Thanks [@ecPablo](https://github.com/ecPablo)! - fix proposal analyzer render issues with array details

## 0.74.2

### Patch Changes

- [#656](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/656) [`bdf4104`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bdf410480189a3d2e568478d61c57b7bc45d1b5a) Thanks [@friedemannf](https://github.com/friedemannf)! - Bump CTF to v0.12.6

## 0.74.1

### Patch Changes

- [#653](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/653) [`173d35e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/173d35ed67760c432bbd4d9886b28089be05aa4f) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(jd): keep wsrpc field as storage

  Looks like WSRPC field cant be removed completely for now as Chainlink repo uses WSRPC field of the JDConfig as temporary storage for lookup later, it requires a refactor on the Chainlink side to address this, in the mean time to unblock the removal of wsrpc in the CLD, we temporary restore the storage functionality of the field.

## 0.74.0

### Minor Changes

- [#643](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/643) [`ade5b2c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ade5b2cc3ed79cd28903da8a7c9e507db977a479) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(JD): remove WSRPC field from JDConfig

  The WSRPC in JDConfig was never needed as it was never used. Only GRPC field is needed.

### Patch Changes

- [#649](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/649) [`fea4ff3`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fea4ff3d632cb2ec0f5affeb61a13240c8a0736e) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(JD): restore WSRPC field to help with graceful migration in chainlink repo and CLD repo

## 0.73.0

### Minor Changes

- [#647](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/647) [`e76e685`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e76e685b8885108f6de1cd2e1d0aed9aa238a2d4) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(jd): new mapper function for chain family

  Maps JD proto ChainType to the chain selector family string

- [#637](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/637) [`fba3c78`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fba3c78eee1ace74883373a114e202dc65ca7063) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(operations): introduce RegisterOperationRelaxed

### Patch Changes

- [#639](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/639) [`724f6f9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/724f6f9d9d38a534c4b6ca386db506c3b4ec1fc6) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(pipeline): remove support for object format for payload in input yaml file

## 0.72.0

### Minor Changes

- [#633](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/633) [`006c70a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/006c70afd9aa8fd5f6e7cb66bea41740e5f0d9b2) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - feat(mcms): fetch pipeline PR data before decoding a proposal

### Patch Changes

- [#634](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/634) [`143bdc3`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/143bdc3f1cc3fe707b6427cc64d8fc447812c4e2) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - patch: update rpc regex for anvil to include tailscale urls

- [#630](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/630) [`f0ede8e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f0ede8ef3fb91f4cdec1c2061ae478093314c84f) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): use proposalContextProvider in mcmsv2's get-op-count and is-timelock-done commands

## 0.71.4

### Patch Changes

- [#625](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/625) [`ea28b23`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ea28b239855994124c1b44a7fe3073fee364cb82) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(catalog): load from catalog when mode is all

## 0.71.3

### Patch Changes

- [#620](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/620) [`ac4ad05`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ac4ad054866ef21d77ac457fa0bc1b4606621ba7) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): re-fetch mcm opCount on "txNonce too large" error

## 0.71.2

### Patch Changes

- [#618](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/618) [`e67e1e0`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e67e1e0a1ae228445a2534d61035623b29aa5426) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: abort execute-chain if an operation fails due to txNonce too large

## 0.71.1

### Patch Changes

- [#615](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/615) [`8045d1f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8045d1f47dd99c4d7cc5fba253c2ea5aeab4ac6a) Thanks [@jkongie](https://github.com/jkongie)! - Bump `golang.org/x/crypto` to v0.45.0

- [#616](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/616) [`4430663`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4430663c1c2e93bbb88ff74f26edb5de42405946) Thanks [@jkongie](https://github.com/jkongie)! - Override `js-yaml` to use version v3.14.2

## 0.71.0

### Minor Changes

- [#604](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/604) [`c0b7401`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c0b740129be9c1c5f20f38899a7afd154dc75eed) Thanks [@ecPablo](https://github.com/ecPablo)! - add error decode command

## 0.70.0

### Minor Changes

- [#605](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/605) [`49d9309`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/49d9309e2d629466172547a13ec3f0bb066ac024) Thanks [@jadepark-dev](https://github.com/jadepark-dev)! - add TON container with config option

### Patch Changes

- [#607](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/607) [`b56241c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b56241c94059bba6f1074ba1a2ccc2b7cb2eb5a3) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: allow empty payload field in input file

## 0.69.0

### Minor Changes

- [#600](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/600) [`bf1ed32`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/bf1ed32d94eb4e5d600b114b992098bafd09983b) Thanks [@jkongie](https://github.com/jkongie)! - Adds a new load option `WithChains` for loading the test engine environment

  This option is useful if you want to manually construct and configure chains before adding
  them to the environment instead of using the existing predefined chain loader options.

## 0.68.2

### Patch Changes

- [#588](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/588) [`d1febae`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d1febae4f3ac585643b4e42c38ac0bed0b05f7e8) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): make proposalContextProvider a required param of BuildMCMSv2Cmd

- [#595](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/595) [`95f96d9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/95f96d9b2cd55a6fbbadd0728f2dd072d6fb88c2) Thanks [@finleydecker](https://github.com/finleydecker)! - Added -eth flag to the evm nodes fund command. Example: users can now use "-eth 10" to fund nodes up to 10 eth. Also added a new line separator to the current balance log to improve readability.

- [#594](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/594) [`5936b3b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5936b3b6c1bde3f9b4ec9ed85e0adfe8b3fd84e4) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): improve error handling in confirmTransaction

## 0.68.1

### Patch Changes

- [#596](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/596) [`2d45dcc`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2d45dccaa26f3d88654c018f4c88398f6e15b893) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: set sui default image to mysten/sui-tools:devnet-v1.61.0

- [#598](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/598) [`c38ded5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c38ded56a3edeab59e780ea6c390b0d1680d3a32) Thanks [@FelixFan1992](https://github.com/FelixFan1992)! - fix(engine/test): default SUI image to CTF Provider

## 0.68.0

### Minor Changes

- [#586](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/586) [`f3a2a36`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f3a2a36fa860430ae86bfe692824205c36b9200c) Thanks [@jkongie](https://github.com/jkongie)! - Updates Test Engine Sui container to use a specific devnet image and generate different deploy keys for each container

### Patch Changes

- [#585](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/585) [`7760d13`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7760d1301f777545f8bbe5a86eb4200d8a34e26c) Thanks [@jadepark-dev](https://github.com/jadepark-dev)! - Clean up TON CTF Provider, update test infra methods

## 0.67.0

### Minor Changes

- [#573](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/573) [`f7a31c2`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f7a31c2b79a846f36b4b6116222c5498b5b3742f) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - add grpc keepalive, retries and connection closure functionality

- [#580](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/580) [`0baab99`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0baab9933ad267632f44ef0ec928cfb25b50481e) Thanks [@jadepark-dev](https://github.com/jadepark-dev)! - expose TON CTF configs to caller

- [#577](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/577) [`a1074b1`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a1074b15fe62936ebd4b19e2941cc9a144416c1d) Thanks [@jkongie](https://github.com/jkongie)! - Updates `go-ethereum` to v1.16.7

- [#579](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/579) [`5d15395`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5d153955f7b8b48f8f36d1585eda337940aaff01) Thanks [@giogam](https://github.com/giogam)! - feat: adds 'all' datastore config option

## 0.66.1

### Patch Changes

- [#574](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/574) [`2077cd6`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2077cd61f5f18d6d9c5c009d30394b4269d1663e) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: handle ports in regexp used to identify public rpc urls

## 0.66.0

### Minor Changes

- [#571](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/571) [`8db262d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8db262d0549378a3565e5445c5180baf8d72b3d0) Thanks [@jkongie](https://github.com/jkongie)! - Bump chain-selectors package to v1.0.81

### Patch Changes

- [#570](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/570) [`eb74395`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/eb743959612d2506ba0888f6e2d0996744ec657b) Thanks [@skudasov](https://github.com/skudasov)! - Adds a new EVM Confirm Functor which allows the user to specify a custom wait interval for checking confirmation.
  Example

  ```golang
  		p, err := cldf_evm_provider.NewRPCChainProvider(
  			d.ChainSelector,
  			cldf_evm_provider.RPCChainProviderConfig{
  				DeployerTransactorGen: cldf_evm_provider.TransactorFromRaw(
  					getNetworkPrivateKey(),
  				),
  				RPCs: []rpcclient.RPC{
  					{
  						Name:               "default",
  						WSURL:              rpcWSURL,
  						HTTPURL:            rpcHTTPURL,
  						PreferredURLScheme: rpcclient.URLSchemePreferenceHTTP,
  					},
  				},
  				ConfirmFunctor: cldf_evm_provider.ConfirmFuncGeth(
  					30*time.Second,
  					// set custom confirm ticker time because Anvil's blocks are instant
  					cldf_evm_provider.WithTickInterval(5*time.Millisecond),
  				),
  			},
  		).Initialize(context.Background())
  ```

## 0.65.0

### Minor Changes

- [#568](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/568) [`109b6f8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/109b6f83cf7363665f02bba420fc149d677fccc0) Thanks [@giogam](https://github.com/giogam)! - feat: adds HMAC authentication support for catalog remote

- [#559](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/559) [`57ee135`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/57ee135753c5e7cabb5c14777d8fdf043f8b90a0) Thanks [@ecPablo](https://github.com/ecPablo)! - Add support to decode proposals that use EIP-1967 proxies

- [#562](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/562) [`aa38817`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/aa38817cb9b737abbc0af8de275521f8b5e5ee06) Thanks [@jkongie](https://github.com/jkongie)! - Removes the import of a root `go.mod` from a scaffolded domain

- [#567](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/567) [`d06057a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d06057a020107659cf0b0e3697b43006bdb784f6) Thanks [@JohnChangUK](https://github.com/JohnChangUK)! - Sui MCMS upgrade

### Patch Changes

- [#530](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/530) [`dc2c113`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/dc2c113025c1c22f1384e33b1a10535df4ccfa30) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: make config files and chain credentials optional

## 0.64.0

### Minor Changes

- [#556](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/556) [`0e60a11`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0e60a11bcc6dee70dfd01d3cf89027935f53082c) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - feat(mcms): check MCM state before calling SetRoot or Execute

### Patch Changes

- [#565](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/565) [`ba781b4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ba781b4ba8a7744546f4b6132b18f0162aca3cad) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): check if anvil config is valid after selecting the rpc

## 0.63.0

### Minor Changes

- [#560](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/560) [`ed679a7`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ed679a72ab740098c18ad1750c80d81782e12d95) Thanks [@huangzhen1997](https://github.com/huangzhen1997)! - bump sui binding version

### Patch Changes

- [#561](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/561) [`280ce37`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/280ce37cdcc0981e55168ba099f909720d5912e1) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: computation of the txNonce attribute in the upf converter

## 0.62.0

### Minor Changes

- [#555](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/555) [`844d9d3`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/844d9d31b29fbc6122a9ef23e7216bdebc2a1bbe) Thanks [@amit-momin](https://github.com/amit-momin)! - Specified Solana image to enable arm64 runtime architecture

## 0.61.0

### Minor Changes

- [#552](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/552) [`32b13c5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/32b13c5964a6d9234bb0342140ff8ca14c36da79) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat: add catalog service integration for datastore operations

  Features:

  - Add catalog service support for datastore management as alternative to local file storage
  - Add `MergeMigrationDataStoreCatalog` method for catalog-based datastore persistence
  - Existing `MergeMigrationDataStore` method continues to work for file-based storage (no breaking changes)
  - Add unified `MergeDataStoreToCatalog` function for both initial migration and ongoing merge operations
  - All catalog operations are transactional to prevent data inconsistencies
  - Add `DatastoreType` configuration option (`file`/`catalog`) in domain.yaml to control storage backend
  - Add new CLI command `datastore sync-to-catalog` for initial migration from file-based to catalog storage in CI
  - Add `SyncDataStoreToCatalog` method to sync entire local datastore to catalog
  - CLI automatically selects the appropriate merge method based on domain.yaml configuration
  - Catalog mode does not modify local files - all updates go directly to the catalog service

  Configuration:

  - Set `datastore: catalog` in domain.yaml to enable catalog mode
  - Set `datastore: file` or omit the setting to use traditional file-based storage
  - CLI commands automatically detect the configuration and use the appropriate storage backend

- [#549](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/549) [`3e33b93`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/3e33b93b9b99f25dfb25ad38f0baf4815245da2d) Thanks [@jkongie](https://github.com/jkongie)! - Improve JD Memory client to be aligned with the Job Distributor implementation

## 0.60.1

### Patch Changes

- [#553](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/553) [`8d6a9f7`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8d6a9f72b7a32dabf41e744b8f90fcfd55d0c960) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): select public rpc for fork tests

- [#551](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/551) [`cf4de66`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/cf4de6687f910b5d2c9f02094414902be4d55644) Thanks [@giogam](https://github.com/giogam)! - fix(catalog): updates errors in remote implementation

## 0.60.0

### Minor Changes

- [#541](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/541) [`909e6f4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/909e6f4f60f7bf96feff049550ec43fbcf31bd73) Thanks [@ecPablo](https://github.com/ecPablo)! - refactor proposal analyzer to add a text renderer and move templates to separate files

- [#535](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/535) [`b7c8d06`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b7c8d06de2e0ccc1918fb2c5195a9ea7e78833b4) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(engine): load catalog into datastore

### Patch Changes

- [#547](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/547) [`2229818`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2229818268aaf0d37346b54f464708259b48578d) Thanks [@jkongie](https://github.com/jkongie)! - Bump chain-selectors to v1.0.77

- [#534](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/534) [`9572077`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/95720772c2866478571b1550f4eb557a7b6e2264) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(catalog): use enum instead of string

## 0.59.1

### Patch Changes

- [#544](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/544) [`ea1859a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ea1859abb1f93028dd4421912a8a624190743083) Thanks [@friedemannf](https://github.com/friedemannf)! - Bump CTF to v0.11.3

## 0.59.0

### Minor Changes

- [#536](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/536) [`d35d8de`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d35d8dee7937065b91945995b9f19c621d4111d5) Thanks [@jkongie](https://github.com/jkongie)! - JD Memory Client now supports filtering in `ListNodes`

- [#542](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/542) [`5b3a421`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5b3a4215b61ff2c4abdfe8da449a21cedb23f87e) Thanks [@jkongie](https://github.com/jkongie)! - Aligns MemoryJobDistributor `ProposeJob` and `RevokeJob` to have the same functionality as the JobDistributor service

- [#540](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/540) [`35d9189`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/35d9189e56bcf6ebfbecd91d5f482b48fa2555a3) Thanks [@jkongie](https://github.com/jkongie)! - JD Memory Client now supports filtering in `ListJobs`

## 0.58.1

### Patch Changes

- [#532](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/532) [`f2a3b3e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f2a3b3e574c1e01c83013cd4704b9b5773e95481) Thanks [@vyzaldysanchez](https://github.com/vyzaldysanchez)! - Display nodes versions when running the `jd node list` command.

## 0.58.0

### Minor Changes

- [#514](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/514) [`406cb82`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/406cb82e83841d6bd59ea51f30dd42d9c063b5e8) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Handle Catalog ResponseStatus errors as grpc errors

- [#518](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/518) [`99ee634`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/99ee634219c860412cbcbb1c72ee54665591fe8d) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(offchain): new JD in memory client

- [#522](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/522) [`0cbed61`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0cbed61c14a6e42259251908584d0cf072a19a8c) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(engine): integrate memory JD to test runtime

- [#529](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/529) [`4bb40fa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4bb40fa07966170287ca272bb4b3f8f30eefeb99) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(catalog): new datastore field in domain.yaml

  Field `datastore` is introduced to configure in future where should the data be written to, either file(json) - current behaviour or remote on the catalog service.
  By default, this field will be set to file for backwards compatibility.

- [#520](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/520) [`2cc6462`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2cc6462f255a46ab560bb2f2c26e29c2da59f378) Thanks [@ecPablo](https://github.com/ecPablo)! - improve decoded proposal error to use bullets instead of tables

- [#517](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/517) [`5220e9a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5220e9ac21cdeea6b9e1bb67e8a0e5f96118d30e) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(engine/test): support memory catalog

### Patch Changes

- [#527](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/527) [`8041f81`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8041f810b501341e0b0650ad266079acf11b79e3) Thanks [@giogam](https://github.com/giogam)! - chore: remove Catalog field from Environment struct

- [#524](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/524) [`41b8c65`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/41b8c65ad3f443f8591f9be842c090a888305fce) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: remove dep on chainlink-common

## 0.57.0

### Minor Changes

- [#482](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/482) [`62ed5d0`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/62ed5d05c11f7307e58c7dc1cc057ea22188229e) Thanks [@rodrigombsoares](https://github.com/rodrigombsoares)! - Implement SUI proposal analyzer

- [#510](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/510) [`8fd65fe`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8fd65fe3e8e7964cd95b00b077ab32e5175552c6) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: enable strict yaml unmarshalling

  When unmarshalling from yaml input for pipelines, if there is a field not defined in the struct, an error will be returned. This helps catch typos and misconfigurations early.

- [#512](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/512) [`c035859`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c03585984f44c1a9e4adea94770844c3e6defbfd) Thanks [@jkongie](https://github.com/jkongie)! - Adds a new option to test engine environment loading for setting NodeIDs

  `WithNodeIDs` - option to set NodeIDs into the test environment

### Patch Changes

- [#507](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/507) [`3ba8202`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/3ba820224cce3188b32fa4ff14653761fb5033c7) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(catalog/memory): convert to true inmemory implementation

  Remove dependency on pgtest

- [#515](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/515) [`e458ad5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e458ad5956c9bcae29b2c1628dfd43e630276041) Thanks [@vyzaldysanchez](https://github.com/vyzaldysanchez)! - Add extra info(workflow key and p2p key bundles) when logging JD nodes on a table format.

## 0.56.0

### Minor Changes

- [#503](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/503) [`08dcbfb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/08dcbfb55c4f98a887194435cb9b353fa4e79f1c) Thanks [@jkongie](https://github.com/jkongie)! - Loading the test engine environment now accepts two new options

  - `WithDatastore` - Allows setting a custom datastore for the environment.
  - `WithAddressBook` - Allows setting a custom address book for the environment.

- [#505](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/505) [`602bf03`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/602bf03fc196692616dd90a5861bf69326b50781) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - refactor(catalog): rename MemoryDatastore

  NewMemoryDataStore -> NewMemoryCatalogDataStore

- [#508](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/508) [`5761e5a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5761e5aa9264d8bed071a65be03c1b364d8d4932) Thanks [@jkongie](https://github.com/jkongie)! - Adds a new option to test engine environment loading for setting an offchain client.

  - `WithOffchainClient` option to set an offchain client into the test environment

### Patch Changes

- [#504](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/504) [`4a11d81`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4a11d81939b8f465dce99626002a683cfd082c65) Thanks [@ecPablo](https://github.com/ecPablo)! - improve error display for cases where revert has no data using tracing with anvil and searching for revert reasons on abi registry

## 0.55.1

### Patch Changes

- [#499](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/499) [`57c4e9b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/57c4e9b919e37d36950e111e9d5a0fa4b7c59cd9) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(catalog/memory): remove unused MemoryDataStoreConfig

- [#502](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/502) [`5abc4df`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5abc4dfca2f8556bc9bf737c1961f8b1accbddf3) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - Add extra alias for Sui deployer key env var

## 0.55.0

### Minor Changes

- [#492](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/492) [`7243af8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7243af8be1d93101d0655c3424d60f6dcdcd883e) Thanks [@jkongie](https://github.com/jkongie)! - update aptos dep to v1.9.1

- [#474](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/474) [`fdcf28d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fdcf28d6dfaa06c3bdfd92acae2c5e414479c4af) Thanks [@ecPablo](https://github.com/ecPablo)! - add predecessors and opcount calculation logic to proposalutils package.

### Patch Changes

- [#496](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/496) [`fea372c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fea372c81f6cd0a49c1793cd3f87686842669a1d) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: update db controller to accept context

- [#498](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/498) [`ce51cbe`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ce51cbefa3d92af9fa91bb5a6dcb531d69b76f54) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: anvil env should check for addresses from DataStore as well

- [#495](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/495) [`126609e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/126609e1400ac56142d751291ec9cab83d716216) Thanks [@ecPablo](https://github.com/ecPablo)! - get delay for advancing time from the proposal instead of constant value

- [#497](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/497) [`976d232`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/976d232b8ea5263f743f2885d50cbd18a5712b48) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(catalog/memory): remove dependency on testing.T

## 0.54.1

### Patch Changes

- [#489](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/489) [`63fda69`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/63fda69d0c909cbb4cc6104a68cb03a92f4b12be) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - feat(mcms): query Timelock contract for CallProxy address

- [#478](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/478) [`f318c97`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f318c973efa79278e12d2fa3c1b4eb52daa178bf) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: restore owner after mcm.SetConfig() in fork tests with --test-signer

## 0.54.0

### Minor Changes

- [#481](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/481) [`1f7f6bc`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1f7f6bc9be80a9522680022868448847b62ba20a) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(operations): introduce AsUntypedRelaxed

### Patch Changes

- [#484](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/484) [`fb9d9bf`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fb9d9bfef7456052e4c07fc4bba3c9b16e5b4bd5) Thanks [@jkongie](https://github.com/jkongie)! - Fixes test engine MCMS execution when multiple proposals have the same `validUntil` timestamp.

  A salt override is added to each timelock proposal persisted to the state to ensure unique operation
  IDs in test environments where multiple proposals may have identical timestamps. This salt is used
  in the hashing algorithm to determine the root of the merkle tree.

- [#479](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/479) [`930e469`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/930e469560608b9ec32a398393862b9fbc4d663a) Thanks [@jkongie](https://github.com/jkongie)! - Fixes MCMS Execution failing to Set Root in the test engine

- [#475](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/475) [`8d9ded3`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8d9ded34541c2f47211832cfbbb7906b40c5746f) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix(mcms): check for PostOpCountReached errors in Solana as well

## 0.53.0

### Minor Changes

- [#469](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/469) [`a24665b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/a24665bd8fdf430e66ba5157902826e05002e080) Thanks [@jkongie](https://github.com/jkongie)! - Adds support for Sui and Tron chains in the test engine

- [#473](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/473) [`d2bdd22`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d2bdd22c0f0948c8c2c00707d20e7f0f634e5d91) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: add --changeset-index flag for array format YAML files

  Add support for running changesets by index position to handle duplicate
  changeset names in array format input files.

- [#467](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/467) [`fe7e75b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fe7e75b20c5369e2566408848c387d40efa56a6d) Thanks [@jkongie](https://github.com/jkongie)! - Adds a new test engine task to sign and execute all pending proposals

  A new test engine runtime task has been added to improve the experience
  of signing and executing MCMS proposals. This new task will sign and
  execute all pending proposals that previous ChangesetTasks have generated.

  ```
  signingKey, _ := crypto.GenerateKey() // Use your actual MCMS signing key here instead
  runtime.Exec(
      SignAndExecuteProposalsTask([]*ecdsa.PrivateKey{signingKey},
  )
  ```

## 0.52.0

### Minor Changes

- [#463](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/463) [`aba39dc`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/aba39dcef98a36a296ae8794c5cbdd3cf1763225) Thanks [@finleydecker](https://github.com/finleydecker)! - Bump chain-selectors

## 0.51.0

### Minor Changes

- [#429](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/429) [`1703535`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/17035351f97836c3ac9b21bc9aa08c68be602c1f) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Adding Jira package

### Patch Changes

- [#459](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/459) [`98c0ebc`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/98c0ebcd969a96c8026df5f4328040026ae8051b) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: preserve large integers in YAML to JSON conversion

  Fixes TestSetDurablePipelineInputFromYAML_WithPathResolution by preventing
  large integers from being converted to scientific notation during JSON
  marshaling, which causes issues when unmarshaling to big.Int.

  **Problem:**

  - YAML parsing converts large numbers like `2000000000000000000000` to `float64(2e+21)`
  - JSON marshaling converts `float64(2e+21)` to scientific notation `"2e+21"`
  - big.Int cannot unmarshal scientific notation, causing errors

## 0.50.1

### Patch Changes

- [#456](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/456) [`4b10eea`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4b10eea297fbf759da9c3b1586bd2bc58a78387c) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - disable jd in fork env

## 0.50.0

### Minor Changes

- [#452](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/452) [`41464d4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/41464d42dae680365ae303f8b75ed5483abd30a2) Thanks [@jkongie](https://github.com/jkongie)! - Add `runtime.New()` convenience function for runtime initialization

  Provides a simpler way to create runtime instances using functional options for environment configuration.

- [#445](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/445) [`967a01b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/967a01b612f313b6dadf9defc8d1cafad9cb9927) Thanks [@jkongie](https://github.com/jkongie)! - Adds tasks to the test engine runtime to sign and execute MCMS proposals

- [#451](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/451) [`0e64684`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0e646842a1b0b2d65d06d649b3893d1508dfe223) Thanks [@jkongie](https://github.com/jkongie)! - Adds new convenience method `environment.New` to the test engine to bring up a new test environment

  The `environment.New` method is a wrapper around the environment loading struct and allows the user
  to load a new environment without having to instantiate the `Loader` struct themselves.

  The `testing.T` argument has been removed and it's dependencies have been replaced with:

  - A `context.Context` argument to the `Load` and `New` functions
  - A new functional option `WithLogger` which overrides the default noop logger.

  While this is a breaking change, the test environment is still in development and is not in actual usage yet.

### Patch Changes

- [#454](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/454) [`d87d8ef`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d87d8ef1a5bfc9b20ee981636ea8e7ea7992922a) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Bump CTF to fix docker security dependency

- [#455](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/455) [`4788ba4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/4788ba47adabdf3de0a89d44dbcf14440fc4feec) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: update ValidUntil when running "mcmsv2 reset-proposal"

## 0.49.1

### Patch Changes

- [#425](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/425) [`5583eba`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5583ebabf916d5188b6e21c4ae35c4ac44b2b462) Thanks [@giogam](https://github.com/giogam)! - feat(environment): use network config chains instead of addressbook

## 0.49.0

### Minor Changes

- [#437](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/437) [`2224427`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/22244276bcb7c192a81530e6d4434f371684fbb6) Thanks [@jkongie](https://github.com/jkongie)! - **[BREAKING]** Refactored `LoadOffchainClient` to use functional options

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

  ```

- [#428](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/428) [`e172683`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/e172683ff4f28c79ed865a4224e8e2e04b0953e8) Thanks [@jkongie](https://github.com/jkongie)! - Adds a test engine runtime for executing changesets in unit/integration tests

- [#443](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/443) [`9e6bc1d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/9e6bc1dcbb3803fc4c85794b194c08224a073ada) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: introduce template-input command for generating YAML input

  This commit introduces a new template-input command that generates YAML input templates from Go struct types for durable pipeline changesets. The command uses reflection to analyze changeset input types and produces well-formatted YAML templates with type comments to guide users in creating valid input files.

- [#440](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/440) [`7f1af5d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/7f1af5d0a3514f80aec08c3bab29a2ac4276b340) Thanks [@RodrigoAD](https://github.com/RodrigoAD)! - add support for sui in mcms commands

## 0.48.2

### Patch Changes

- [#435](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/435) [`d8a740e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d8a740e8e9d044994d33158c7423091c3f45e137) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(OnlyLoadChainsFor)!: remove migration name parameter for environment option

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

## 0.48.1

### Patch Changes

- [#430](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/430) [`b90b6e5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b90b6e5698be831cb2d36490ad268bd9eec9058a) Thanks [@jkongie](https://github.com/jkongie)! - Fixes dry run Job Distributor being used by default

## 0.48.0

### Minor Changes

- [#424](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/424) [`c241756`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c2417566058ff4dd502a17d9b28242e26968406a) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: enhance OnlyLoadChainsFor to support loading no chains when no chains is provided, eg OnlyLoadChainsFor()

- [#408](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/408) [`2861467`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/286146723b3c9e1b5dccdebdf28eb67af8737cfd) Thanks [@jkongie](https://github.com/jkongie)! - Adds the ability to load an environment in a test engine. This is intended for use in unit and integration tests.

- [#421](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/421) [`de7bd86`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/de7bd8630bba8aab219b8c7d46b37e8d546633f1) Thanks [@giogam](https://github.com/giogam)! - feat(datastore): require DataStore in environment Load

## 0.47.0

### Minor Changes

- [#410](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/410) [`deda430`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/deda430af37545dfee2c69618fa7931525411a49) Thanks [@ecPablo](https://github.com/ecPablo)! - Add CLI command to reset proposals

- [#405](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/405) [`f8dab56`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/f8dab5616f77830495a12c318c4cd5a9017c1ca5) Thanks [@jkongie](https://github.com/jkongie)! - [BREAKING] Simplifies the function signature of `environment.Load` and `environment.LoadForkedEnvironment`

## 0.46.0

### Minor Changes

- [#411](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/411) [`8d4e755`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8d4e7550c77d7c321f4b7f07c62e78bc161d6b04) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat: add ctf geth provider

### Patch Changes

- [#417](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/417) [`c53af0e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c53af0e083223063e657c9911ded5fce11a9ab98) Thanks [@giogam](https://github.com/giogam)! - chore: removes ocr type aliases from deployment package

- [#416](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/416) [`c72eaff`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c72eaff8f972c030a90be15ec164d5992153ec2a) Thanks [@friedemannf](https://github.com/friedemannf)! - Bump CTF to v0.10.24

- [#418](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/418) [`181501a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/181501a738507fadd278158f0e6b8742cef2fd1d) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: update findWorkspaceRoot to not check for root go.mod

## 0.45.2

### Patch Changes

- [#409](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/409) [`b3bd891`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b3bd8917d6d188c861cbefa36575647eb0d54849) Thanks [@gustavogama-cll](https://github.com/gustavogama-cll)! - fix: embed Anvil's MCMS layout file instead of loading it from the filesystem

- [#396](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/396) [`d79b3c0`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d79b3c080458ea41b6d69d6149dff37ecf791a9f) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(scaffold): sanitize env name for go package name

## 0.45.1

### Patch Changes

- [#292](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/292) [`42dc440`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/42dc440931f27d2dffdf95f95048e155222d045e) Thanks [@stackman27](https://github.com/stackman27)! - fixes for sui provider

## 0.45.0

### Minor Changes

- [#403](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/403) [`ba78126`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ba78126e1050c7d9a6a788d43e9a58e439eda07f) Thanks [@ecPablo](https://github.com/ecPablo)! - port MCMS v2 CLI commands from CLD repo

## 0.44.0

### Minor Changes

- [#391](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/391) [`35282b9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/35282b99b3fbba58729ba47395f11c1adc682222) Thanks [@jadepark-dev](https://github.com/jadepark-dev)! - TON provider - liteserver connection string support

- [#386](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/386) [`8102d0e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8102d0ef1c94e6a52421b8ce415094631c7770bc) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(chain): add support for converting string address to bytes for each chain

### Patch Changes

- [#399](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/399) [`96b85a9`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/96b85a9918202e5b9ce577e75c73032e1a361a04) Thanks [@giogam](https://github.com/giogam)! - chore: moves durable-pipelines commands

- [#397](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/397) [`90fe2a8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/90fe2a8c29631bcd9599524f9540b3d03bc8578e) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: migrate migration command from CLD

## 0.43.0

### Minor Changes

- [#382](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/382) [`0de5d03`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0de5d036ff3761856a7f5d61a48c01b7c7275ca9) Thanks [@jkongie](https://github.com/jkongie)! - Remove DeployerSeed field from Tron and Ton Chain

- [#384](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/384) [`6a9e263`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6a9e2636cc24690b0ec54be064c631756d4e02e7) Thanks [@ecPablo](https://github.com/ecPablo)! - Adds proposal analyzer package to experimental

## 0.42.0

### Minor Changes

- [#373](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/373) [`14af85c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/14af85cf91cab7cb971c22285ac7f894de04fab1) Thanks [@nicolasgnr](https://github.com/nicolasgnr)! - Adding TON support to CLD

- [#381](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/381) [`2b56b8a`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2b56b8a3edf78bc9891cff6c9922e66b0f9d2b5f) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - feat: migrate jd commands to framework

- [#379](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/379) [`237e390`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/237e390a86ad8fd000bf59ed4ef8fa5a65438630) Thanks [@jkongie](https://github.com/jkongie)! - Private Key Generators for Chain Providers no longer expose the private key as a public field

### Patch Changes

- [#375](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/375) [`0b48f42`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0b48f4218fcb025c1d9f43f27e5b4fb3ead70c9d) Thanks [@giogam](https://github.com/giogam)! - chore: moves ocr_secrets deployment -> offchain/ocr

- [#374](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/374) [`cc451a0`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/cc451a060d31af1ebec445a871d0de5957e80a9c) Thanks [@giogam](https://github.com/giogam)! - chore(deployment): removes multiclient aliases

- [#370](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/370) [`b978df4`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b978df4abac430dcbb1655a246e5a16f882bf96a) Thanks [@ajaskolski](https://github.com/ajaskolski)! - feat: migrate evm commands from cld

## 0.41.0

### Minor Changes

- [#369](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/369) [`208aac1`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/208aac10a18740a99179a7d6aff5bd753901eee8) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat(command): migrate env command

- [#364](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/364) [`12c9d4d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/12c9d4d52865b8bd50a7157e071e75d1326ac249) Thanks [@jkongie](https://github.com/jkongie)! - Remove migration file from the scaffold. Pipelines is the preferred way to run Changesets.

- [#368](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/368) [`9b3255e`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/9b3255e41dfd3104703256da0bbdf890f78365fc) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Migrate fork environment and anvil to framework

### Patch Changes

- [#365](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/365) [`0c50737`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0c5073749981e6cf478a0465590efff88d249abb) Thanks [@giogam](https://github.com/giogam)! - chore: duplicates multiclient and rpc_config in chain/evm

## 0.40.0

### Minor Changes

- [#363](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/363) [`6fa7125`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6fa71251f39b58037fff33f4dda6c65a6e9851fe) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Add load environment to framework

### Patch Changes

- [#360](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/360) [`5bd5575`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5bd557547d295f2fc8012ccb8924d1b1d0da5ec8) Thanks [@giogam](https://github.com/giogam)! - feat: updates domains scaffold

## 0.39.0

### Minor Changes

- [#357](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/357) [`8289afa`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8289afa366c63d9288251b4c355f65135537b517) Thanks [@jkongie](https://github.com/jkongie)! - Moves `environment.Config` to `config.Config`

- [#352](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/352) [`5088d9c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/5088d9c99c8bdae6f00d4190ef8f0b1766394cb6) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix: migrate Update Node from CLD

### Patch Changes

- [#355](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/355) [`63d8f65`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/63d8f65e3f156862d00ed4562fbdc9d460b8a1e7) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Fix flaky test in TestRegisterNode_Success_Plugin

- [#358](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/358) [`6ef6875`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6ef6875d8d498f55c9bc89d6969c20006297cc71) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - Migrate LoadCatalogStore and LoadJDClient to engine

- [#356](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/356) [`fd159c8`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/fd159c8e15ae31aacefeb9dea7b5c947c97b5ad6) Thanks [@giogam](https://github.com/giogam)! - feat(environment): removes getLegacyNetworkTypes

## 0.38.0

### Minor Changes

- [#349](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/349) [`811b2b5`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/811b2b5b0d60614e3d3df4e7e96d88282b4e35d5) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Add scaffolding to the framework

- [#350](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/350) [`0344f12`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/0344f129ce2b6d4f6af1df2abbb1c52763667fa3) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - feat: migrate ProposeJob and RegisterNode from CLD to CLDF

- [#347](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/347) [`6639d28`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/6639d285c03ab8e92ca649508c5bba0fe22e2e51) Thanks [@jkongie](https://github.com/jkongie)! - Remove unused `changeset.RequireDeployerKeyBalance` method as it is unused.

- [#351](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/351) [`2587c25`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/2587c254390f7a0e9f50aace28b12ea8b6878223) Thanks [@jkongie](https://github.com/jkongie)! - Adds an `environment.LoadConfig` method to load all config for an environment

## 0.37.1

### Patch Changes

- [#338](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/338) [`d207c3b`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d207c3b4f9bd9a6adf8171cb333e325c45588bd6) Thanks [@giogam](https://github.com/giogam)! - feat: load network types from domain config

- [#344](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/344) [`cf1cc45`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/cf1cc4586605d13276edb2033fdd68942e78946a) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(operations): handle missing case causing Operations APi unable to serialize certain Marshalable struct

- [#345](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/345) [`ba11ea0`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ba11ea024b8d8796aa834dbab2b60b977f3b0595) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - Migrated Dry Run JD Client from CLD

## 0.37.0

### Minor Changes

- [#341](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/341) [`365c01c`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/365c01c097e1d165953b51ff3c67bebe087f15fb) Thanks [@jkongie](https://github.com/jkongie)! - [Breaking] Change CognitoAuth field names

- [#335](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/335) [`1602a8d`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/1602a8d105c74a1d0070ff1617c05b9e2ab3cac6) Thanks [@DimitriosNaikopoulos](https://github.com/DimitriosNaikopoulos)! - Add changeset registry

## 0.36.0

### Minor Changes

- [#336](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/336) [`ed5dc34`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/ed5dc34958880d38cfa0c48e24181dfeaf8fd4f0) Thanks [@jkongie](https://github.com/jkongie)! - Adds a struct to generate Cognito oauth tokens for Job Distributor

- [#340](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/340) [`c9e6857`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/c9e6857091dd0a761e2c469dc3c60f8cf8551f60) Thanks [@jkongie](https://github.com/jkongie)! - Adds `cli` package to `engine/cld/legacy`

### Patch Changes

- [#327](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/327) [`8d0cbfb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/8d0cbfb6581d6611609a08d50414506086221514) Thanks [@giogam](https://github.com/giogam)! - feat: adds domain config

- [#328](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/328) [`240a44f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/240a44f7d17afb9343ca09a402cbe60d2e9c0fd7) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - fix(sui): use correct docker image on arm64

- [#332](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/332) [`d577271`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/d57727139630cd31da5cfa4c4d74d8174dd2e347) Thanks [@ajaskolski](https://github.com/ajaskolski)! - refactor: move environment network from cld

- [#337](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/337) [`b3bdffc`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/b3bdffcd0346d329fc4e6a37397dec4cf277bc27) Thanks [@giogam](https://github.com/giogam)! - chore: refactor camelCase with snake_case

## 0.35.0

### Minor Changes

- [#324](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/324) [`399f4bb`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/399f4bb7e12093888dec713b2f043cd471f3e30c) Thanks [@bytesizedroll](https://github.com/bytesizedroll)! - Changeset framework to CLDF

## 0.34.1

### Patch Changes

- [#319](https://github.com/smartcontractkit/chainlink-deployments-framework/pull/319) [`34bcf7f`](https://github.com/smartcontractkit/chainlink-deployments-framework/commit/34bcf7f2f5e8b90a4297ef457f3dcb31e95a16f3) Thanks [@graham-chainlink](https://github.com/graham-chainlink)! - add prefix to proposal if timestamp is available

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
