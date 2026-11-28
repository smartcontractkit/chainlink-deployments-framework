package balance

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/samber/lo"
	"golang.org/x/sync/errgroup"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
)

type Status string

const (
	StatusOK           Status = "ok"
	StatusLow          Status = "low"
	StatusCritical     Status = "critical"
	StatusUnconfigured Status = "unconfigured" // no threshold for this chain
	StatusUnsupported  Status = "unsupported"  // no Reader for this family
	StatusError        Status = "error"
)

// Threshold values are in human units of the native token (e.g. "0.5" ETH).
type Threshold struct {
	WarnBelow     *big.Int `json:"warn_below,omitempty"     yaml:"warn_below"`
	CriticalBelow *big.Int `json:"critical_below,omitempty" yaml:"critical_below"`
}

type Thresholds map[uint64]Threshold // by chain selector

type Result struct {
	ChainSelector    uint64 `json:"chain_selector"`
	ChainName        string `json:"chain_name"`
	Family           string `json:"family"`
	Address          string `json:"address,omitempty"`
	Balance          string `json:"balance,omitempty"`           // base units (wei, lamports, ...)
	BalanceFormatted string `json:"balance_formatted,omitempty"` // human units
	WarnBelow        string `json:"warn_below,omitempty"`
	CriticalBelow    string `json:"critical_below,omitempty"`
	Status           Status `json:"status"`
	Message          string `json:"message,omitempty"`
}

// Reader knows how to read the signer address and native balance for one chain family.
type Reader interface {
	Family() string
	Decimals(common.BlockChain) uint8
	Address(common.BlockChain) (string, error)
	Balance(context.Context, common.BlockChain) (*big.Int, error)
}

// ---- options -------------------------------------------------------------

type config struct {
	readers          map[string]Reader
	timeout          time.Duration
	concurrency      int
	warnGasLimit     *big.Int
	criticalGasLimit *big.Int
}

var (
	defaultConcurrency      = 16
	defaultTimeout          = 15 * time.Second
	defaultReaders          = []Reader{&evm.BalanceReader{} /* solanaReader, aptosReader, */}
	defaultWarnGasLimit     = big.NewInt(200_000)
	defaultCriticalGasLimit = big.NewInt(50_000)
)

type CheckOption func(*config)

func WithReaders(rs ...Reader) CheckOption {
	return func(c *config) {
		for _, r := range rs {
			c.readers[r.Family()] = r
		}
	}
}
func WithTimeout(d time.Duration) CheckOption     { return func(c *config) { c.timeout = d } }
func WithConcurrency(n int) CheckOption           { return func(c *config) { c.concurrency = n } }
func WithWarnGasLimit(n *big.Int) CheckOption     { return func(c *config) { c.warnGasLimit = n } }
func WithCriticalGasLimit(n *big.Int) CheckOption { return func(c *config) { c.criticalGasLimit = n } }

// Check reads the signer balance on every chain and classifies it against thresholds.
// It never returns an error: per-chain failures are reported as StatusError.
//
// IMPORTANT: chains must hold the real signers — do NOT pass BlockChains.ReadOnly(),
// which replaces deployer keys with random ones.
func Check(ctx context.Context, chains []common.BlockChain, thresholds Thresholds, opts ...CheckOption) []Result {
	cfg := config{
		readers:          lo.KeyBy(defaultReaders, func(r Reader) string { return r.Family() }),
		timeout:          defaultTimeout,
		concurrency:      defaultConcurrency,
		warnGasLimit:     defaultWarnGasLimit,
		criticalGasLimit: defaultCriticalGasLimit,
	}
	for _, o := range opts {
		o(&cfg)
	}

	results := make([]Result, len(chains))
	g := errgroup.Group{} // not WithContext: one chain failing must not cancel the others
	g.SetLimit(cfg.concurrency)

	for i, bc := range chains {
		g.Go(func() error {
			threshold, exists := thresholds[bc.ChainSelector()]
			if !exists {
				threshold = estimateThresholds(ctx, bc, cfg.warnGasLimit, cfg.criticalGasLimit)
				fmt.Printf("ESTIMATED THRESHOLDS for %s: warn=%v, critical=%v\n", bc.String(), threshold.WarnBelow, threshold.CriticalBelow)
			} else {
				fmt.Printf("GIVEN THRESHOLDS for %s: warn=%v, critical=%v\n", bc.String(), threshold.WarnBelow, threshold.CriticalBelow)
			}

			results[i] = checkOne(ctx, cfg, bc, threshold)

			return nil
		})
	}
	_ = g.Wait()

	return results
}

func checkOne(ctx context.Context, cfg config, bc common.BlockChain, th Threshold) (res Result) {
	res = Result{
		ChainSelector: bc.ChainSelector(),
		ChainName:     bc.Name(),
		Family:        bc.Family(),
		WarnBelow:     th.WarnBelow.String(),
		CriticalBelow: th.CriticalBelow.String(),
	}
	defer func() { // a buggy reader must never break the workflow
		if p := recover(); p != nil {
			res.Status, res.Message = StatusError, fmt.Sprintf("panic reading balance: %v", p)
		}
	}()

	reader, ok := cfg.readers[bc.Family()]
	if !ok {
		res.Status, res.Message = StatusUnsupported, "no balance reader for family "+bc.Family()
		return res
	}

	addr, err := reader.Address(bc)
	if err != nil {
		res.Status, res.Message = StatusError, "address: "+err.Error()
		return res
	}
	res.Address = addr

	cctx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()
	bal, err := reader.Balance(cctx, bc)
	if err != nil {
		res.Status, res.Message = StatusError, "balance: "+err.Error()
		return res
	}

	dec := reader.Decimals(bc)
	res.Balance = bal.String()
	res.BalanceFormatted = formatUnits(bal, dec)
	res.Status, res.Message = classify(bal, th)

	return res
}

func classify(bal *big.Int, th Threshold) (Status, string) {
	if th.WarnBelow == nil && th.CriticalBelow == nil {
		if bal.Sign() == 0 {
			return StatusCritical, "balance is zero"
		}

		return StatusUnconfigured, "no threshold configured"
	}
	if th.CriticalBelow != nil && bal.Cmp(th.CriticalBelow) < 0 {
		return StatusCritical, "balance below critical threshold " + th.CriticalBelow.String()
	}
	if th.WarnBelow != nil && bal.Cmp(th.WarnBelow) < 0 {
		return StatusLow, "balance below warning threshold " + th.WarnBelow.String()
	}

	return StatusOK, ""
}

func estimateThresholds(
	ctx context.Context, chain common.BlockChain, warnGasLimit *big.Int, criticalGasLimit *big.Int,
) Threshold {
	evmChain, ok := chain.(evm.Chain)
	if !ok {
		return Threshold{} // only EVM chains are supported for now
	}

	header, err := evmChain.Client.HeaderByNumber(ctx, nil)
	if err != nil {
		return Threshold{}
	}

	if header.BaseFee == nil {
		return Threshold{}
	}
	if header.BaseFee.Cmp(big.NewInt(0)) == 0 {
		gasPrice, err := evmChain.Client.SuggestGasPrice(ctx)
		if err != nil {
			return Threshold{}
		}
		header.BaseFee = gasPrice
	}

	return Threshold{
		WarnBelow:     new(big.Int).Mul(warnGasLimit, header.BaseFee),
		CriticalBelow: new(big.Int).Mul(criticalGasLimit, header.BaseFee),
	}
}

// ---- typed adapter: casts BlockChain -> concrete chain type -------------

// var errNoSigner = errors.New("chain has no signer configured")
// // TypedReader turns per-family closures over a concrete chain type C into a Reader.
// type TypedReader[C any] struct {
// 	family   string
// 	decimals func(C) uint8
// 	address  func(C) (string, error)
// 	balance  func(context.Context, C) (*big.Int, error)
// }
//
// func (r TypedReader[C]) Family() string { return r.family }
//
// func (r TypedReader[C]) Decimals(bc common.BlockChain) uint8 {
// 	c, _ := as[C](bc)
// 	return r.decimals(c)
// }
//
// func (r TypedReader[C]) Address(bc cldf_chain.BlockChain) (string, error) {
// 	c, err := as[C](bc)
// 	if err != nil {
// 		return "", err
// 	}
// 	return r.address(c)
// }
//
// func (r TypedReader[C]) Balance(ctx context.Context, bc cldf_chain.BlockChain) (*big.Int, error) {
// 	c, err := as[C](bc)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return r.balance(ctx, c)
// }
//
// // as handles chains stored either as values or as pointers in BlockChains.
// func as[C any](bc cldf_chain.BlockChain) (C, error) {
// 	switch c := any(bc).(type) {
// 	case C:
// 		return c, nil
// 	case *C:
// 		if c != nil {
// 			return *c, nil
// 		}
// 	}
// 	var zero C
// 	return zero, fmt.Errorf("unexpected chain type %T for %s", bc, bc.String())
// }
//
// ```
//
// The unit helpers (`parseUnits` and `formatUnits`) are about 20 lines using `big.Rat`: `"0.5"` × 10^dec gives base units, and the reverse direction uses `FloatString(dec)` with trailing zeros trimmed. `cmpUint64` is a trivial comparison helper.
//
// ### Where the inputs come from at the call sites

// ```go
// thresholds: built once from the networks config that was already loaded
// func ThresholdsFromNetworks(nets *cfgnet.Config) Thresholds // reads the new Network.DeployerBalance field
//
// // CLDF engine/cld/commands/pipeline/run.go, just before registry.Apply (env.BlockChains has real keys)
// if path := os.Getenv("CLD_BALANCE_REPORT_PATH"); path != "" {
// 	results := balance.Check(ctx, env.BlockChains, balance.ThresholdsFromNetworks(cfg.Networks))
// 	balance.Log(lggr, results)
// 	if err := balance.WriteJSON(path, results); err != nil {
// 		lggr.Warnw("failed to write balance report", "err", err) // warn only
// 	}
// }
//
// // mcms-tools `proposal check-balance`: same call with chains from blockchainsDS.Load(selectors)
// ```
//
// ### Why it's shaped this way
// - **The registry is keyed by `Family()`.** `Check` never mentions `evm.Chain` or any other concrete type. Adding a family means writing one `typedReader` and adding it to `DefaultReaders()`.
// - **The cast lives in `typedReader`.** Each family's closures get the concrete type with its `Client` and signer, without repeating the value-or-pointer handling.
// - **Every outcome is a `Result`.** A missing reader, a nil signer, an RPC timeout or even a panic produce `unsupported` or `error` for that one chain. `Check` never fails, and the other chains are still reported.
// - **Each chain gets its own timeout, with a concurrency limit.** One slow RPC can't stall the run, and 100+ mainnet chains don't open 100+ connections at once.
// - **Testing:** `WithReaders(fakeReader{...})` covers `Check` and `classify` with no RPC. The EVM reader can be tested against the simulated backend.
//
// Two caveats:
// - **zkSync:** zkSync chains also have `DeployerKeyZkSyncVM`. If it ever differs from `DeployerKey`, the EVM reader needs to pick the right one when `IsZkSyncVM` is true.
// - **`chain/gas_estimator.go`:** CLDF already contains an EVM fee estimator there, including the OP-stack L1 data fee. That's useful if we later replace static thresholds with the estimated-cost idea.
//
//
