<div align="center">
  <h1>Chainlink Deployments Framework</h1>
  <a><img src="https://github.com/smartcontractkit/chainlink-deployments-framework/actions/workflows/push-main.yml/badge.svg" /></a>
  <br/>
  <br/>
</div>

This repository contains the Chainlink Deployments Framework, a comprehensive set of libraries that enables developers to build, manage, and execute(in future) deployment changesets.
The framework includes the Operations API and Datastore API.

## Usage

```bash
$ go get github.com/smartcontractkit/chainlink-deployments-framework
```

## Imports

```go
import (
  "github.com/smartcontractkit/chainlink-deployments-framework/deployment" // for writing changesets (migrated from chainlink/deployments
  "github.com/smartcontractkit/chainlink-deployments-framework/operations" // for operations API
  "github.com/smartcontractkit/chainlink-deployments-framework/datastore" // for datastore API
)
```

## Development

### Installing Dependencies

Install the required tools using [asdf](https://asdf-vm.com/guide/getting-started.html):

```bash
asdf install
```

### Linting

```bash
task lint
```

### Testing

```bash
task test
```

## How to add a new Chain

To add a new chain to the framework, follow these steps:

1. **Create the chain structure and implementation:**

   - Create a new directory under `chain/` with the name of your chain (e.g., `chain/newchain/`).
   - Create a new `Chain` struct that embeds the `ChainMetadata` struct
   - See the Sui or TON implementations as reference examples. EVM, Solana, and Aptos chains follow a different implementation pattern as they were added before CLDF.

2. **Implement chain providers:**

   - Create a `provider/` subdirectory under your chain directory (e.g., `chain/newchain/provider/`)
   - Implement one or more provider types that satisfy the `chain.Provider` interface:
     ```go
     type Provider interface {
         Initialize(ctx context.Context) (BlockChain, error)
         Name() string
         ChainSelector() uint64
         BlockChain() BlockChain
     }
     ```
   - Common provider types to implement:
     - **RPC Provider**: Connects to a live blockchain node via RPC
     - **Simulated Provider**: Creates an in-memory simulated chain for testing (if needed)
     - **CTF Provider**: Connects to Chainlink Testing Framework environments (if needed)

   Example RPC provider implementation:

   ```go
   package provider

   import (
       "context"
       "errors"
       "fmt"

       "github.com/smartcontractkit/chainlink-deployments-framework/chain"
       "github.com/smartcontractkit/chainlink-deployments-framework/chain/newchain"
   )

   // RPCChainProviderConfig holds the configuration for the RPC provider
   type RPCChainProviderConfig struct {
       RPCURL            string
       DeployerSignerGen SignerGenerator // Your chain-specific signer
   }

   func (c RPCChainProviderConfig) validate() error {
       if c.RPCURL == "" {
           return errors.New("rpc url is required")
       }
       if c.DeployerSignerGen == nil {
           return errors.New("deployer signer generator is required")
       }
       return nil
   }

   // Ensure provider implements the interface
   var _ chain.Provider = (*RPCChainProvider)(nil)

   type RPCChainProvider struct {
       selector uint64
       config   RPCChainProviderConfig
       chain    *newchain.Chain
   }

   func NewRPCChainProvider(selector uint64, config RPCChainProviderConfig) *RPCChainProvider {
       return &RPCChainProvider{
           selector: selector,
           config:   config,
       }
   }

   func (p *RPCChainProvider) Initialize(ctx context.Context) (chain.BlockChain, error) {
       if p.chain != nil {
           return p.chain, nil // Already initialized
       }

       if err := p.config.validate(); err != nil {
           return nil, fmt.Errorf("failed to validate config: %w", err)
       }

       // Initialize your chain client
       client, err := newchain.NewClient(p.config.RPCURL)
       if err != nil {
           return nil, fmt.Errorf("failed to create client: %w", err)
       }

       // Generate deployer signer
       signer, err := p.config.DeployerSignerGen.Generate()
       if err != nil {
           return nil, fmt.Errorf("failed to generate signer: %w", err)
       }

       p.chain = &newchain.Chain{
           Selector: p.selector,
           Client:   client,
           Signer:   signer,
           URL:      p.config.RPCURL,
       }

       return *p.chain, nil
   }

   func (p *RPCChainProvider) Name() string {
       return "NewChain RPC Provider"
   }

   func (p *RPCChainProvider) ChainSelector() uint64 {
       return p.selector
   }

   func (p *RPCChainProvider) BlockChain() chain.BlockChain {
       return *p.chain
   }
   ```

3. **Add environment configuration bindings:**

   - Update `engine/cld/config/env/config.go`:
     - Add a new config struct for your chain (e.g., `NewChainConfig`):
       ```go
       // NewChainConfig is the configuration for the NewChain chains.
       //
       // WARNING: This data type contains sensitive fields and should not be logged or set in file
       // configuration.
       type NewChainConfig struct {
           DeployerKey string `mapstructure:"deployer_key" yaml:"deployer_key"` // Secret: The private key of the deployer account.
           // Add any other chain-specific configuration fields
       }
       ```
     - Add the new config to the `OnchainConfig` struct:
       ```go
       type OnchainConfig struct {
           // ... existing fields ...
           NewChain NewChainConfig `mapstructure:"newchain" yaml:"newchain"`
       }
       ```
     - Add environment variable bindings in the `envBindings` map:
       ```go
       envBindings = map[string][]string{
           // ... existing bindings ...
           "onchain.newchain.deployer_key": {"ONCHAIN_NEWCHAIN_DEPLOYER_KEY"},
           // Add other bindings as needed
       }
       ```

4. **Create chain loader in the CLD engine:**

   - Update `engine/cld/chains/chains.go`:

     - Add a new chain loader struct and methods:

       ```go
       // chainLoaderNewChain implements the ChainLoader interface for NewChain.
       type chainLoaderNewChain struct {
           *baseChainLoader
       }

       // newChainLoaderNewChain creates a new chain loader for NewChain.
       func newChainLoaderNewChain(
           networks *cfgnet.Config, cfg cfgenv.OnchainConfig,
       ) *chainLoaderNewChain {
           return &chainLoaderNewChain{
               baseChainLoader: newBaseChainLoader(networks, cfg),
           }
       }

       // Load loads a NewChain Chain for a selector.
       func (l *chainLoaderNewChain) Load(ctx context.Context, selector uint64) (fchain.BlockChain, error) {
           network, err := l.getNetwork(selector)
           if err != nil {
               return nil, err
           }

           rpcURL := network.RPCs[0].HTTPURL
           c, err := newchainprov.NewRPCChainProvider(selector,
               newchainprov.RPCChainProviderConfig{
                   RPCURL:            rpcURL,
                   DeployerSignerGen: newchainprov.AccountGenPrivateKey(l.cfg.NewChain.DeployerKey),
               },
           ).Initialize(ctx)
           if err != nil {
               return nil, fmt.Errorf("failed to initialize NewChain chain %d: %w", selector, err)
           }

           return c, nil
       }
       ```

     - Add the loader to the `newChainLoaders` function with appropriate configuration checks:

       ```go
       func newChainLoaders(...) map[string]ChainLoader {
           // ... existing loaders ...

           if cfg.NewChain.DeployerKey != "" {
               loaders[chainsel.FamilyNewChain] = newChainLoaderNewChain(networks, cfg)
           } else {
               lggr.Warn("Skipping NewChain chains, no private key found in secrets")
           }

           return loaders
       }
       ```

     - Add interface compliance verification:
       ```go
       var (
           // ... existing verifications ...
           _ ChainLoader = &chainLoaderNewChain{}
       )
       ```

5. **Update chain registry:**

   - Update `chain/blockchain.go`:
     - Add `var _ BlockChain = newchain.Chain{}` at the top to verify interface compliance
     - Create a new getter method (e.g., `NewChainChains()`) that returns `map[uint64]newchain.Chain` (e.g., `NewSuiChains()`)

6. **Update MCMS adapter:**

   - Update `chain/mcms/adapters/chain_access.go`:
     - Import your chain's RPC client library and CLDF chain package:
       ```go
       import (
           // ... existing imports ...
           newchainclient "github.com/your-chain/client-library" // Your chain's RPC client
           cldfnewchain "github.com/smartcontractkit/chainlink-deployments-framework/chain/newchain"
       )
       ```
     - Add your chain's getter method to the `ChainsFetcher` interface:
       ```go
       type ChainsFetcher interface {
           // ... existing methods ...
           NewChainChains() map[uint64]cldfnewchain.Chain
       }
       ```
     - Add a client accessor method to the `ChainAccessAdapter`:
       ```go
       // NewChainClient returns the underlying NewChain RPC client for the given selector.
       func (a *ChainAccessAdapter) NewChainClient(selector uint64) (*newchainclient.Client, bool) {
           ch, ok := a.inner.NewChainChains()[selector]
           if !ok {
               return nil, false
           }

           // Return the RPC client from your Chain type.
           // This assumes your Chain struct has a Client field.
           // Adjust to use the appropriate field or method accessor (e.g., ch.RPCClient, ch.GetClient()).
           return ch.Client, true
       }
       ```
   - Update `chain/mcms/adapters/chain_access_test.go`:
     - Import your chain package and add test coverage for the client accessor method
     - Example test assertion:
       ```go
       gotNewChain, ok := a.NewChainClient(newChainSel)
       require.True(t, ok)
       require.Nil(t, gotNewChain) // or assert on expected client properties
       ```

7. **Write comprehensive tests:**
   - Test chain instantiation
   - Test all interface methods
   - Test the getter method in BlockChains
   - Test provider initialization and chain creation
   - Test provider interface compliance with `var _ chain.Provider = (*YourProvider)(nil)`
   - Test chain loader functionality

**Using Providers:**
Once you've implemented a provider, users can create and initialize chains like this:

```go
// Create provider with configuration
provider := provider.NewRPCChainProvider(chainSelector, provider.RPCChainProviderConfig{
    RPCURL:            "https://your-chain-rpc-url.com",
    DeployerSignerGen: yourSignerGenerator,
})

// Initialize the chain
ctx := context.Background()
blockchain, err := provider.Initialize(ctx)
if err != nil {
    return fmt.Errorf("failed to initialize chain: %w", err)
}
```

## Contributing

For instructions on how to contribute to `chainlink-deployments-framework` and the release process,
see [CONTRIBUTING.md](https://github.com/smartcontractkit/chainlink-deployments-framework/blob/main/CONTRIBUTING.md)

## Releasing

For instructions on how to release `chainlink-deployments-framework`,
see [RELEASE.md](https://github.com/smartcontractkit/chainlink-deployments-framework/blob/main/RELEASE.md)
