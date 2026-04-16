package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/smartcontractkit/chainlink-testing-framework/framework"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/components/blockchain"
	"github.com/smartcontractkit/chainlink-testing-framework/framework/pods"
)

// defaultSuiImage is the mysten/sui-tools image used when CTFChainProviderConfig.Image is unset
// (non–darwin/arm64 hosts). Keep aligned with the Sui release you support in tests.
const defaultSuiImage = "mysten/sui-tools:devnet-v1.69.0"

// demuxDockerExecOutput converts Docker exec attach output to plain text when it uses the
// multiplexed stream format (first byte 1=stdout / 2=stderr). Must run before stripping 0x01,
// which appears in stream headers and would corrupt the stream if removed globally.
func demuxDockerExecOutput(raw string) string {
	if len(raw) == 0 {
		return raw
	}
	if raw[0] != 1 && raw[0] != 2 {
		return raw
	}
	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, strings.NewReader(raw)); err != nil {
		return raw
	}

	return stdout.String() + stderr.String()
}

// parseSuiKeytoolGenerateJSON extracts a SuiWalletInfo from `sui keytool generate --json` output.
// The CLI may print a preamble, and v1.69+ may emit compact one-line JSON; CTF's parser assumes a
// legacy layout (newline after '{') and corrupts compact output.
func parseSuiKeytoolGenerateJSON(keyOut string) (*blockchain.SuiWalletInfo, error) {
	text := demuxDockerExecOutput(keyOut)
	s := strings.ReplaceAll(text, "\x00", "")
	for i := range len(s) {
		if s[i] != '{' {
			continue
		}
		var key blockchain.SuiWalletInfo
		dec := json.NewDecoder(bytes.NewReader([]byte(s[i:])))
		if err := dec.Decode(&key); err != nil {
			continue
		}
		if key.SuiAddress != "" {
			return &key, nil
		}
	}

	return nil, fmt.Errorf("failed to parse SuiWalletInfo from keytool output: %.200q", keyOut)
}

func generateKeyDataCTF(ctx context.Context, containerName, keyCipherType string) (*blockchain.SuiWalletInfo, error) {
	dc, err := framework.NewDockerClient()
	if err != nil {
		return nil, err
	}

	// Ensure a valid Sui client config exists. `sui start --force-regenesis`
	// creates its config under /root/.sui/sui_config/ but the client.yaml it
	// generates may not exist yet when this runs, so we use `sui client --yes`
	// with an explicit config flag to force creation.
	initCmd := []string{"sui", "client", "--client.config", "/root/.sui/sui_config/client.yaml", "--yes", "envs"}
	if initOut, initErr := dc.ExecContainerWithContext(ctx, containerName, initCmd); initErr != nil {
		framework.L.Warn().Err(initErr).Str("out", initOut).Msg("sui client init returned error (may be harmless)")
	}

	cmdStr := []string{"sui", "keytool", "generate", keyCipherType, "--json"}
	keyOut, err := dc.ExecContainerWithContext(ctx, containerName, cmdStr)
	if err != nil {
		return nil, err
	}
	key, err := parseSuiKeytoolGenerateJSON(keyOut)
	if err != nil {
		return nil, fmt.Errorf("%w (raw output: %.300q)", err, keyOut)
	}

	framework.L.Info().Interface("Key", key).Msg("Test key")

	return key, nil
}

func defaultSuiCTF(in *blockchain.Input) {
	if in.Image == "" {
		in.Image = defaultSuiImage
	}
	if in.Port == "" {
		in.Port = blockchain.DefaultSuiNodePort
	}
	if in.FaucetPort == "" {
		in.FaucetPort = blockchain.DefaultFaucetPortNum
	}
}

// newSuiCTFBlockchainNetwork mirrors CTF blockchain.newSui but uses generateKeyDataCTF so
// mysten/sui-tools v1.69.x keytool JSON parses correctly.
func newSuiCTFBlockchainNetwork(ctx context.Context, in *blockchain.Input) (*blockchain.Output, error) {
	defaultSuiCTF(in)
	containerName := framework.DefaultTCName("blockchain-node")

	var files []testcontainers.ContainerFile
	if in.ContractsDir != "" {
		absPath, err := filepath.Abs(in.ContractsDir)
		if err != nil {
			return nil, err
		}
		files = []testcontainers.ContainerFile{
			{
				HostFilePath:      absPath,
				ContainerFilePath: "/",
			},
		}
	}

	containerPort := blockchain.DefaultSuiNodePort + "/tcp"

	imagePlatform := "linux/amd64"
	if in.ImagePlatform != nil && *in.ImagePlatform != "" {
		imagePlatform = *in.ImagePlatform
	}

	if pods.K8sEnabled() {
		return nil, errors.New("K8s support is not yet implemented")
	}

	req := testcontainers.ContainerRequest{
		Image:        in.Image,
		ExposedPorts: []string{containerPort, blockchain.DefaultFaucetPort},
		Name:         containerName,
		Labels:       framework.DefaultTCLabels(),
		Networks:     []string{framework.DefaultNetworkName},
		NetworkAliases: map[string][]string{
			framework.DefaultNetworkName: {containerName},
		},
		HostConfigModifier: func(h *container.HostConfig) {
			h.PortBindings = nat.PortMap{
				nat.Port(containerPort): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: in.Port,
					},
				},
				nat.Port(blockchain.DefaultFaucetPort): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: in.FaucetPort,
					},
				},
			}
			framework.ResourceLimitsFunc(h, in.ContainerResources)
		},
		ImagePlatform: imagePlatform,
		Env: map[string]string{
			"RUST_LOG": "off,sui_node=info",
		},
		Cmd: []string{
			"sui",
			"start",
			"--force-regenesis",
			"--with-faucet",
		},
		Files:      files,
		WaitingFor: wait.ForListeningPort(blockchain.DefaultFaucetPort).WithStartupTimeout(1 * time.Minute).WithPollInterval(200 * time.Millisecond),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	suiAccount, err := generateKeyDataCTF(ctx, containerName, "ed25519")
	if err != nil {
		return nil, err
	}
	if err := fundAccount(fmt.Sprintf("http://%s:%s", "127.0.0.1", in.FaucetPort), suiAccount.SuiAddress); err != nil {
		return nil, err
	}

	return &blockchain.Output{
		UseCache:            true,
		Type:                in.Type,
		Family:              blockchain.FamilySui,
		ContainerName:       containerName,
		Container:           c,
		NetworkSpecificData: &blockchain.NetworkSpecificData{SuiAccount: suiAccount},
		Nodes: []*blockchain.Node{
			{
				ExternalHTTPUrl: fmt.Sprintf("http://%s:%s", host, in.Port),
				InternalHTTPUrl: fmt.Sprintf("http://%s:%s", containerName, blockchain.DefaultSuiNodePort),
			},
		},
	}, nil
}
