---
"chainlink-deployments-framework": patch
---

Adds a new EVM Confirm Functor which allows the user to specify a custom wait interval for checking confirmation.
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
