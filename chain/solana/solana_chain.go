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

	"github.com/gagliardetto/solana-go"
	solRpc "github.com/gagliardetto/solana-go/rpc"

	solCommonUtil "github.com/smartcontractkit/chainlink-ccip/chains/solana/utils/common"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain/internal"
)

const (
	ProgramIDPrefix      = "Program Id: "
	BufferIDPrefix       = "Buffer: "
	SolDefaultCommitment = solRpc.CommitmentConfirmed
)

// Chain represents a Solana chain.
type Chain struct {
	Selector uint64

	// RPC client
	Client *solRpc.Client
	URL    string
	WSURL  string
	// TODO: raw private key for now, need to replace with a more secure way
	DeployerKey *solana.PrivateKey
	Confirm     func(instructions []solana.Instruction, opts ...solCommonUtil.TxModifier) error

	// deploy uses the solana CLI which needs a keyfile
	KeypairPath  string
	ProgramsPath string
}

func (c Chain) CloseBuffers(logger logger.Logger, buffer string) error {
	baseArgs := []string{
		"program",
		"close",
		buffer,                     // buffer address e.g. "5h2npsKHzGpiibLZvKnr12yC31qzvQESRnfovofL4WE3"
		"--keypair", c.KeypairPath, // deployer keypair
		"--url", c.URL, // rpc url
	}

	cmd := exec.Command("solana", baseArgs...) // #nosec G204
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
		cmd = exec.Command("solana", baseArgs...) // #nosec G204
	} else {
		// Keypairs wont be created for devenvs
		logger.Infow("Deploying new program",
			"programFile", programFile)
		cmd = exec.Command("solana", baseArgs...) // #nosec G204
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

	return parseProgramID(output, prefix)
}

func (c Chain) GetAccountDataBorshInto(ctx context.Context, pubkey solana.PublicKey, accountState interface{}) error {
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

// ChainSelector returns the chain selector of the chain
func (c Chain) ChainSelector() uint64 {
	return c.Selector
}

// String returns chain name and selector "<name> (<selector>)"
func (c Chain) String() string {
	return internal.ChainBase{Selector: c.Selector}.String()
}

// Name returns the name of the chain
func (c Chain) Name() string {
	return internal.ChainBase{Selector: c.Selector}.Name()
}
