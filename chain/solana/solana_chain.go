package solana

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sollib "github.com/gagliardetto/solana-go"
	solrpc "github.com/gagliardetto/solana-go/rpc"
	chainsel "github.com/smartcontractkit/chain-selectors"
	solCommonUtil "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"

	chaincommon "github.com/smartcontractkit/chainlink-deployments-framework/chain/internal/common"
	"github.com/smartcontractkit/chainlink-deployments-framework/chain/solana/provider/rpcclient"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

const (
	ProgramIDPrefix      = "Program Id: "
	BufferIDPrefix       = "Buffer: "
	SolDefaultCommitment = solrpc.CommitmentConfirmed
)

// Chain represents a Solana chain.
type Chain struct {
	Selector uint64

	// RPC client
	Client *solrpc.Client
	URL    string
	WSURL  string
	// DeployerKey is the deployer's raw private key. It is nil when the deployer signs remotely
	// (e.g. with AWS KMS). Prefer Deployer(), which works in both cases.
	DeployerKey *sollib.PrivateKey

	// DeployerSigner signs transactions on behalf of the deployer. If DeployerKey is nil, the
	// keypair at KeypairPath (if any) is a separate key used only by the Solana CLI.
	DeployerSigner Signer

	// SendAndConfim provides a utility function to send a transaction and waits for confirmation.
	// Options can be passed to modify the transaction such as setting additional signers or
	// modifying compute unit limits.
	SendAndConfirm func(
		ctx context.Context, instructions []sollib.Instruction, opts ...rpcclient.TxModifier,
	) error

	// Legacy: Use SendAndConfirm instead. This function will be removed in the future.
	// Confirm is a legacy function that sends a transaction and waits for confirmation.
	Confirm func(instructions []sollib.Instruction, opts ...solCommonUtil.TxModifier) error

	// deploy uses the solana CLI which needs a keyfile
	KeypairPath  string
	ProgramsPath string
}

// Deployer returns the signer for the deployer account, or nil if the chain has no deployer.
func (c Chain) Deployer() Signer {
	if c.DeployerSigner != nil {
		return c.DeployerSigner
	}
	if c.DeployerKey != nil {
		return NewPrivateKeySigner(*c.DeployerKey)
	}

	return nil
}

// hasRemoteDeployer reports whether the deployer signs remotely (e.g. with AWS KMS), in which
// case the keypair at KeypairPath is a separate key used only by the Solana CLI.
func (c Chain) hasRemoteDeployer() bool {
	return c.DeployerKey == nil && c.DeployerSigner != nil
}

func (c Chain) CloseBuffers(logger logger.Logger, buffer string) error {
	if c.hasRemoteDeployer() {
		return c.closeBufferWithDeployer(logger, buffer)
	}

	baseArgs := []string{
		"program",
		"close",
		buffer,                     // buffer address e.g. "5h2npsKHzGpiibLZvKnr12yC31qzvQESRnfovofL4WE3"
		"--keypair", c.KeypairPath, // deployer keypair
		"--url", c.URL, // rpc url
		"--bypass-warning",
	}

	cmd := exec.CommandContext(context.TODO(), "solana", baseArgs...) // #nosec G204
	logger.Infof("Closing buffers with command: %s", cmd.String())

	// Capture the command output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		logger.Errorw("Error closing buffers",
			"error", err,
			"stdout", stdout.String(),
			"stderr", stderr.String())

		return err
	}
	logger.Infow("Closed buffers",
		"stdout", stdout.String(),
		"stderr", stderr.String())

	return nil
}

// ProgramInfo contains information about a Solana program.
type ProgramInfo struct {
	Name  string
	Bytes int
}

// Overallocate should be set when deploying any program that may eventually be owned by timelock
// Overallocate is mutually exclusive with isUpgrade
func (c Chain) DeployProgram(logger logger.Logger, programInfo ProgramInfo, isUpgrade bool, overallocate bool) (string, error) {
	if c.KeypairPath == "" {
		return "", errors.New("deploying programs requires a local keypair for the Solana CLI, but none is configured")
	}

	programName := programInfo.Name
	programFile := filepath.Join(c.ProgramsPath, programName+".so")
	if _, err := os.Stat(programFile); err != nil {
		return "", fmt.Errorf("program file not found: %w", err)
	}
	programKeyPair := filepath.Join(c.ProgramsPath, programName+"-keypair.json")

	cliCommand := "deploy"
	prefix := ProgramIDPrefix
	if isUpgrade {
		cliCommand = "write-buffer"
		prefix = BufferIDPrefix
	}

	// Base command with required args
	baseArgs := []string{
		"program", cliCommand,
		programFile,                // .so file
		"--keypair", c.KeypairPath, // deployer keypair
		"--url", c.URL, // rpc url
		"--use-rpc", // use rpc for deployment
	}

	var cmd *exec.Cmd
	// We need to specify the program ID on the initial deploy but not on upgrades
	// Upgrades happen in place so we don't need to supply the keypair
	// It will write the .so file to a buffer and then deploy it to the existing keypair
	if !isUpgrade {
		logger.Infow("Deploying program with existing keypair",
			"programFile", programFile,
			"programKeyPair", programKeyPair)
		baseArgs = append(baseArgs, "--program-id", programKeyPair)
		programBytes := programInfo.Bytes
		if overallocate && programBytes > 0 {
			baseArgs = append(baseArgs, "--max-len", strconv.Itoa(programBytes))
		}
		cmd = exec.CommandContext(context.TODO(), "solana", baseArgs...) // #nosec G204
	} else {
		// Keypairs wont be created for devenvs
		logger.Infow("Deploying new program",
			"programFile", programFile)
		cmd = exec.CommandContext(context.TODO(), "solana", baseArgs...) // #nosec G204
	}

	// Capture the command output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error deploying program: %s: %s", err.Error(), stderr.String())
	}

	// Parse and return the program ID
	output := stdout.String()

	// TODO: obviously need to do this better
	time.Sleep(5 * time.Second)

	id, err := parseProgramID(output, prefix)
	if err != nil {
		return "", err
	}

	// The Solana CLI cannot sign with a remote deployer, so the program was deployed with the
	// local CLI keypair. Hand the upgrade (or buffer) authority over to the deployer so that the
	// rest of the deployment, which signs as the deployer, can manage it.
	if c.hasRemoteDeployer() {
		if err := c.transferAuthorityToDeployer(logger, id, isUpgrade); err != nil {
			return "", err
		}
	}

	return id, nil
}

// transferAuthorityToDeployer sets the deployer as the upgrade authority of a program, or as the
// buffer authority of a buffer when isBuffer is true. It signs with the local CLI keypair, which
// is the current authority after a CLI deploy.
func (c Chain) transferAuthorityToDeployer(logger logger.Logger, address string, isBuffer bool) error {
	deployer := c.DeployerSigner.PublicKey().String()

	var args []string
	if isBuffer {
		args = []string{
			"program", "set-buffer-authority", address,
			"--new-buffer-authority", deployer,
			"--keypair", c.KeypairPath, "--url", c.URL,
		}
	} else {
		args = []string{
			"program", "set-upgrade-authority", address,
			"--new-upgrade-authority", deployer,
			// The deployer cannot sign through the CLI, so the CLI cannot check its signature.
			"--skip-new-upgrade-authority-signer-check",
			"--keypair", c.KeypairPath, "--url", c.URL,
		}
	}

	cmd := exec.CommandContext(context.TODO(), "solana", args...) // #nosec G204
	logger.Infow("Transferring authority to deployer", "address", address, "deployer", deployer, "isBuffer", isBuffer)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error transferring authority of %s to deployer %s: %s: %s",
			address, deployer, err.Error(), stderr.String())
	}

	return nil
}

// closeBufferWithDeployer closes a program buffer whose authority is the deployer, sending the
// rent to the deployer. It is used instead of the Solana CLI when the deployer signs remotely.
func (c Chain) closeBufferWithDeployer(logger logger.Logger, buffer string) error {
	bufferKey, err := sollib.PublicKeyFromBase58(buffer)
	if err != nil {
		return fmt.Errorf("invalid buffer address %q: %w", buffer, err)
	}

	deployer := c.DeployerSigner.PublicKey()
	ix := sollib.NewInstruction(
		sollib.BPFLoaderUpgradeableProgramID,
		sollib.AccountMetaSlice{
			sollib.Meta(bufferKey).WRITE(),
			sollib.Meta(deployer).WRITE(),
			sollib.Meta(deployer).SIGNER(),
		},
		// UpgradeableLoaderInstruction::Close, bincode encoded as a little-endian u32.
		[]byte{5, 0, 0, 0},
	)

	logger.Infow("Closing buffer with deployer", "buffer", buffer, "deployer", deployer)
	if err := c.SendAndConfirm(context.TODO(), []sollib.Instruction{ix}); err != nil {
		logger.Errorw("Error closing buffer", "buffer", buffer, "error", err)

		return err
	}
	logger.Infow("Closed buffer", "buffer", buffer)

	return nil
}

func (c Chain) GetAccountDataBorshInto(ctx context.Context, pubkey sollib.PublicKey, accountState interface{}) error {
	err := solCommonUtil.GetAccountDataBorshInto(ctx, c.Client, pubkey, SolDefaultCommitment, accountState)
	if err != nil {
		return err
	}

	return nil
}

// parseProgramID parses the program ID from the deploy output.
func parseProgramID(output string, prefix string) (string, error) {
	// Look for the program ID in the CLI output
	// Example output: "Program Id: <PROGRAM_ID>"
	startIdx := strings.Index(output, prefix)
	if startIdx == -1 {
		return "", errors.New("failed to find program ID in output")
	}
	startIdx += len(prefix)
	endIdx := strings.Index(output[startIdx:], "\n")
	if endIdx == -1 {
		endIdx = len(output)
	}

	return output[startIdx : startIdx+endIdx], nil
}

// Author note: Have to implement the blockhain interface methods explicitly below
// instead of composing the ChainMetadata struct to avoid breaking change since there are existing usage.

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Name()
}

// Family returns the family of the chain
func (c Chain) Family() string {
	return chaincommon.ChainMetadata{Selector: c.Selector}.Family()
}

// NetworkType returns the type of network the chain is on (e.g. mainnet, testnet)
func (c Chain) NetworkType() (chainsel.NetworkType, error) {
	return chaincommon.ChainMetadata{Selector: c.Selector}.NetworkType()
}

// IsNetworkType checks if the chain is on the given network type
func (c Chain) IsNetworkType(networkType chainsel.NetworkType) bool {
	return chaincommon.ChainMetadata{Selector: c.Selector}.IsNetworkType(networkType)
}

func (c Chain) ReadOnly() (chaincommon.BlockChain, error) {
	privateKey, err := sollib.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key for read-only chain %v: %w", c.Selector, err)
	}
	c.DeployerKey = &privateKey
	c.DeployerSigner = NewPrivateKeySigner(privateKey)

	rpcClient := rpcclient.New(solrpc.New(c.URL), privateKey)
	c.Client = rpcClient.Client

	c.SendAndConfirm = func(ctx context.Context, instructions []sollib.Instruction, opts ...rpcclient.TxModifier) error {
		_, err := rpcClient.SendAndConfirmTx(ctx, instructions, rpcclient.WithTxModifiers(opts...),
			rpcclient.WithRetry(1, 50*time.Millisecond))

		return err
	}

	c.Confirm = func(instructions []sollib.Instruction, opts ...solCommonUtil.TxModifier) error {
		_, err := solCommonUtil.SendAndConfirm(context.Background(), rpcClient.Client, instructions,
			privateKey, solrpc.CommitmentFinalized, opts...)

		return err
	}

	return c, nil
}
