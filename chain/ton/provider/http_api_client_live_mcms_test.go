package provider

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xssnick/tonutils-go/address"

	"github.com/smartcontractkit/chainlink-ton/cciplib/ton/tvm"
	mcmsbind "github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/mcms"
	"github.com/smartcontractkit/chainlink-ton/pkg/bindings/mcms/timelock"
	tonsdk "github.com/smartcontractkit/mcms/sdk/ton"
	"github.com/smartcontractkit/mcms/types"
)

// Read-only checks of the MCMS SDK inspectors and pre-send getters over the HTTP API.
// Enabled by TON_HTTP_V2_URL plus TON_MCMS_ADDRESS and/or TON_TIMELOCK_ADDRESS.

func liveMCMSEnv(t *testing.T, key string) string {
	t.Helper()

	v := os.Getenv(key)
	if v == "" {
		t.Skipf("%s not set", key)
	}

	return v
}

// logRawStack logs the raw v2 stack of a get method, to see exactly how values are encoded.
func logRawStack(t *testing.T, c *httpAPIClient, addr, method string) {
	t.Helper()

	res, err := v2Post[runGetMethodResponse](t.Context(), c, "runGetMethod", runGetMethodRequest{
		Address: addr, Method: method, Stack: [][]string{},
	})
	require.NoError(t, err)
	t.Logf("%s raw: exit_code=%d", method, res.ExitCode)
	for i, e := range res.Stack {
		b, _ := json.Marshal(e)
		if len(b) > 220 {
			b = append(b[:220], "..."...)
		}
		t.Logf("  [%d] %s", i, b)
	}
}

func Test_Live_MCMS(t *testing.T) {
	c := liveClient(t)
	addr := liveMCMSEnv(t, "TON_MCMS_ADDRESS")
	ctx := t.Context()

	logRawStack(t, c, addr, "getOpPendingInfo")

	// Exactly what the executor calls before every SetRoot / Execute.
	blk, err := c.CurrentMasterchainInfo(ctx)
	require.NoError(t, err)
	info, err := tvm.CallGetter(ctx, c, blk, address.MustParseAddr(addr), mcmsbind.GetOpPendingInfo)
	require.NoError(t, err)
	t.Logf("getOpPendingInfo: validAfter=%d opFinalizationTimeout=%d receiver=%v", info.ValidAfter, info.OpFinalizationTimeout, info.OpPendingReceiver)

	insp := tonsdk.NewInspector(c)

	opCount, err := insp.GetOpCount(ctx, addr)
	require.NoError(t, err)
	t.Logf("GetOpCount=%d", opCount)

	root, validUntil, err := insp.GetRoot(ctx, addr)
	require.NoError(t, err)
	t.Logf("GetRoot=%s validUntil=%d", root.Hex(), validUntil)

	md, err := insp.GetRootMetadata(ctx, addr)
	require.NoError(t, err)
	t.Logf("GetRootMetadata: mcm=%s startingOpCount=%d", md.MCMAddress, md.StartingOpCount)

	cfg, err := insp.GetConfig(ctx, addr)
	require.NoError(t, err)
	t.Logf("GetConfig: quorum=%d signers=%d groups=%d", cfg.Quorum, len(cfg.Signers), len(cfg.GroupSigners))
	logRawStack(t, c, addr, "getConfig")

	var walk func(prefix string, g types.Config)
	walk = func(prefix string, g types.Config) {
		t.Logf("%squorum=%d signers=%d subgroups=%d", prefix, g.Quorum, len(g.Signers), len(g.GroupSigners))
		for _, sub := range g.GroupSigners {
			walk(prefix+"  ", sub)
		}
	}
	walk("  ", *cfg)
}

func Test_Live_Timelock(t *testing.T) {
	c := liveClient(t)
	addr := liveMCMSEnv(t, "TON_TIMELOCK_ADDRESS")
	ctx := t.Context()

	logRawStack(t, c, addr, "getOpPendingInfo")

	blk, err := c.CurrentMasterchainInfo(ctx)
	require.NoError(t, err)
	info, err := tvm.CallGetter(ctx, c, blk, address.MustParseAddr(addr), timelock.GetOpPendingInfo)
	require.NoError(t, err)
	t.Logf("timelock getOpPendingInfo: validAfter=%d opFinalizationTimeout=%d", info.ValidAfter, info.OpFinalizationTimeout)

	insp, ok := tonsdk.NewTimelockInspector(c).(*tonsdk.TimelockInspector)
	require.True(t, ok)

	minDelay, err := insp.GetMinDelay(ctx, addr)
	require.NoError(t, err)
	t.Logf("GetMinDelay=%d", minDelay)

	for name, get := range map[string]func() ([]string, error){
		"admins":     func() ([]string, error) { return insp.GetAdmins(ctx, addr) },
		"proposers":  func() ([]string, error) { return insp.GetProposers(ctx, addr) },
		"executors":  func() ([]string, error) { return insp.GetExecutors(ctx, addr) },
		"bypassers":  func() ([]string, error) { return insp.GetBypassers(ctx, addr) },
		"cancellers": func() ([]string, error) { return insp.GetCancellers(ctx, addr) },
	} {
		members, err := get()
		require.NoError(t, err, name)
		t.Logf("%s=%v", name, members)
	}

	var opID [32]byte // an unknown operation: every status must be false
	for name, get := range map[string]func() (bool, error){
		"IsOperation":        func() (bool, error) { return insp.IsOperation(ctx, addr, opID) },
		"IsOperationPending": func() (bool, error) { return insp.IsOperationPending(ctx, addr, opID) },
		"IsOperationReady":   func() (bool, error) { return insp.IsOperationReady(ctx, addr, opID) },
		"IsOperationDone":    func() (bool, error) { return insp.IsOperationDone(ctx, addr, opID) },
	} {
		ok, err := get()
		require.NoError(t, err, name)
		require.False(t, ok, name)
	}
}
