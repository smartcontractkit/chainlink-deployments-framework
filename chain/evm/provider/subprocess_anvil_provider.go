package provider

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/ethclient"
	chainsel "github.com/smartcontractkit/chain-selectors/remote"
	"github.com/smartcontractkit/freeport"

	fchain "github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/gas"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/provider/rpcclient"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

type subprocessAnvilChainProvider struct {
	logger    logger.Logger
	selector  uint64
	url       string
	config    SubprocessAnvilChainProviderConfig
	chain     fchain.BlockChain
	freeports func(int) ([]int, error)
}

type SubprocessAnvilChainProviderConfig struct {
	Command               string
	Args                  []string
	RpcURLs               []string
	DeployerTransactorGen SignerGenerator
	ConfirmFunctor        ConfirmFunctor
	ClientOpts            []func(*rpcclient.MultiClient)
	GasConfig             *gas.Config
	NumAdditionalAccounts uint
}

func (c *SubprocessAnvilChainProviderConfig) validate() error {
	if len(c.RpcURLs) == 0 {
		return errors.New("no rpc URLs provided")
	}
	if c.DeployerTransactorGen == nil {
		return errors.New("no deployer transactor gen provided")
	}

	return nil
}

func NewSubprocessAnvilChainProvider(
	lggr logger.Logger,
	selector uint64,
	config SubprocessAnvilChainProviderConfig,
) *subprocessAnvilChainProvider {
	return &subprocessAnvilChainProvider{
		logger:    lggr,
		selector:  selector,
		config:    config,
		freeports: freeport.Take,
	}
}

func (p *subprocessAnvilChainProvider) Initialize(ctx context.Context) (fchain.BlockChain, error) {
	err := p.config.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid subprocess anvil chain provider config: %w", err)
	}

	chainDetails, err := chainsel.GetChainDetailsBySelector(ctx, p.selector)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain details for selector: %w", err)
	}

	attempt := uint(0)
	maxAttempts := max(uint(len(p.config.RpcURLs)), 5)

	err = retry.Do(func() error {
		return p.execSubprocess(ctx, attempt, chainDetails.ChainID)
	}, retryOptions(ctx, &attempt, maxAttempts)...)
	if err != nil {
		return nil, err
	}

	p.chain, err = newAnvilBlockChain(ctx, p.logger, p.selector, p.url, chainDetails.ChainID,
		p.config.DeployerTransactorGen, p.config.ConfirmFunctor, p.config.NumAdditionalAccounts,
		p.config.ClientOpts, p.config.GasConfig)
	if err != nil {
		return nil, err
	}

	return p.chain, nil
}

func (p *subprocessAnvilChainProvider) execSubprocess(ctx context.Context, attempt uint, chainID string) error {
	ports, err := p.freeports(1)
	if err != nil {
		return fmt.Errorf("failed to get free port: %w", err)
	}

	portStr := strconv.Itoa(ports[0])
	p.url = "http://localhost:" + portStr

	rpcURL := p.config.RpcURLs[attempt%uint(len(p.config.RpcURLs))]

	command := "anvil"
	if p.config.Command != "" {
		command = p.config.Command
	}

	args := append([]string{
		"--host", "127.0.0.1",
		"--port", portStr,
		"--chain-id", chainID,
		"--fork-url", rpcURL,
	}, p.config.Args...)

	p.logger.Infof("running anvil subprocess: %s %s", command, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, command, args...) //nolint:gosec

	err = logSubprocessOutput(p.logger, cmd)
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start anvil subprocess: %w", err)
	}

	err = waitRpcReady(ctx, p.url)
	if err != nil {
		return fmt.Errorf("failed to connect to anvil's rpc: %w", err)
	}

	return nil
}

func (p *subprocessAnvilChainProvider) GetNodeHTTPURL() string {
	return p.url
}

func (p *subprocessAnvilChainProvider) BlockChain() fchain.BlockChain {
	return p.chain
}

func logSubprocessOutput(lggr logger.Logger, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe for anvil subprocess: %w", err)
	}
	stdoutScanner := bufio.NewScanner(stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe for anvil subprocess: %w", err)
	}
	stderrScanner := bufio.NewScanner(stderr)

	go func() {
		for stdoutScanner.Scan() {
			lggr.Infof("[anvil] %s", stdoutScanner.Text())
		}
		if stdoutScanner.Err() != nil {
			lggr.Errorf("failed to read stdout from anvil subprocess: %v", stdoutScanner.Err())
		}
	}()

	go func() {
		for stderrScanner.Scan() {
			lggr.Infof("[anvil][error] %s", stderrScanner.Text())
		}
		if stderrScanner.Err() != nil {
			lggr.Errorf("failed to read stderr from anvil subprocess: %v", stderrScanner.Err())
		}
	}()

	return nil
}

func waitRpcReady(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	connect := func() error {
		client, err := ethclient.DialContext(ctx, url)
		if err != nil {
			return err
		}
		_, err = client.BlockNumber(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			err := connect()
			if err == nil {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func retryOptions(ctx context.Context, attempt *uint, maxAttempts uint) []retry.Option {
	return []retry.Option{
		retry.Context(ctx),
		retry.Attempts(maxAttempts),
		retry.OnRetry(func(a uint, _ error) { *attempt = a + 1 }),
		retry.Delay(1 * time.Second),
		retry.DelayType(retry.FixedDelay),
	}
}
