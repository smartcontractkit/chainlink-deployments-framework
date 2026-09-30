package environment

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/smartcontractkit/chainlink-testing-framework/framework"
)

// rpcHealthChecker probes an RPC URL for liveness/reachability.
type rpcHealthChecker interface {
	Check(ctx context.Context, rpcURL string) error
}

// hostRPCHealthChecker checks RPC reachability from the host process directly. This is correct for
// public RPCs, but gives the wrong answer for URLs that are only reachable from inside the CTF
// docker network (e.g. an anvil fork container's network alias, or another container's
// host-mapped port as seen from a sibling container) -- see containerRPCHealthChecker.
type hostRPCHealthChecker struct{}

func (hostRPCHealthChecker) Check(ctx context.Context, rpcURL string) error {
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to rpc %v: %w", rpcURL, err)
	}
	defer client.Close()

	_, err = client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve block number: %w", err)
	}

	return nil
}

// containerRPCHealthChecker checks RPC reachability from inside the shared CTF docker network,
// by exec-ing curl inside a single long-lived helper container. This matters for anvil fork
// tests: candidate RPC URLs are used as --fork-url by another anvil container, so they must be
// reachable from inside that network, not just from the host running this code. A host-side
// check can wrongly pass a URL (e.g. a container's host-mapped "localhost:<port>") that the
// forking container can't actually reach, while filtering out the network-alias URL that would
// have worked.
//
// The helper container is started lazily on first use and is not safe for concurrent use.
type containerRPCHealthChecker struct {
	once      sync.Once
	container testcontainers.Container
	startErr  error
}

func (h *containerRPCHealthChecker) ensureStarted(ctx context.Context) error {
	h.once.Do(func() {
		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{
				Image:      "curlimages/curl:8.11.0",
				Networks:   []string{framework.DefaultNetworkName},
				Entrypoint: []string{"sleep"},
				Cmd:        []string{"infinity"},
				WaitingFor: wait.ForExec([]string{"curl", "--version"}),
			},
			Started: true,
		})
		if err != nil {
			h.startErr = fmt.Errorf("failed to start rpc health check container: %w", err)
			return
		}
		h.container = container
	})

	return h.startErr
}

func (h *containerRPCHealthChecker) Check(ctx context.Context, rpcURL string) error {
	err := h.ensureStarted(ctx)
	if err != nil {
		return err
	}

	exitCode, out, err := h.container.Exec(ctx, []string{
		"curl", "-s", "-f", "-m", "5", "-X", "POST",
		"-H", "Content-Type: application/json",
		"--data", `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`,
		rpcURL,
	})
	if err != nil {
		return fmt.Errorf("failed to exec rpc health check: %w", err)
	}
	if exitCode != 0 {
		body, _ := io.ReadAll(out)
		return fmt.Errorf("rpc health check failed with exit code %d: %s", exitCode, body)
	}

	return nil
}

// Close terminates the helper container, if one was started.
func (h *containerRPCHealthChecker) Close(ctx context.Context) error {
	if h.container == nil {
		return nil
	}

	return testcontainers.TerminateContainer(h.container, testcontainers.StopContext(ctx))
}
