package provider

import (
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
)

func multiClientOpts(cfg *gas.Config, base []func(*rpcclient.MultiClient)) []func(*rpcclient.MultiClient) {
	opts := append([]func(*rpcclient.MultiClient){}, base...)
	if cfg == nil || cfg.MaxTxGasLimit == 0 {
		return opts
	}

	return append(opts, rpcclient.WithMaxTxGasLimit(cfg.MaxTxGasLimit))
}
