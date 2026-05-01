package stellar_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/stellar/go-stellar-sdk/clients/rpcclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	protocolrpc "github.com/stellar/go-stellar-sdk/protocols/rpc"
	"github.com/stellar/go-stellar-sdk/strkey"
	"github.com/stellar/go-stellar-sdk/txnbuild"
	"github.com/stellar/go-stellar-sdk/xdr"
)

const (
	// defaultTxnTimeBound is the default validity window used when no
	// WithTxnTimeBound option is provided.
	defaultTxnTimeBound = 120 * time.Second

	// minFeeBuffer is the minimum extra stroops added on top of MinResourceFee,
	// used as a floor when the percentage bump (feeBumpFactor) would yield less.
	minFeeBuffer = int64(10_000)
)

// stellarRPCClient abstracts the Soroban RPC methods used by Deployer,
// allowing tests to inject a mock without hitting a real network.
type stellarRPCClient interface {
	SimulateTransaction(ctx context.Context, req protocolrpc.SimulateTransactionRequest) (protocolrpc.SimulateTransactionResponse, error)
	SendTransaction(ctx context.Context, req protocolrpc.SendTransactionRequest) (protocolrpc.SendTransactionResponse, error)
	GetTransaction(ctx context.Context, req protocolrpc.GetTransactionRequest) (protocolrpc.GetTransactionResponse, error)
	GetLedgerEntries(ctx context.Context, req protocolrpc.GetLedgerEntriesRequest) (protocolrpc.GetLedgerEntriesResponse, error)
	GetEvents(ctx context.Context, req protocolrpc.GetEventsRequest) (protocolrpc.GetEventsResponse, error)
}

// DeployerOption configures optional Deployer behaviour.
type DeployerOption func(*Deployer)

// WithAutoRestore controls whether the Deployer automatically submits a
// RestoreFootprint transaction when simulation indicates that persistent
// ledger entries have expired. Enabled by default.
func WithAutoRestore(enabled bool) DeployerOption {
	return func(d *Deployer) { d.autoRestore = enabled }
}

// WithFeeBumpFactor sets the multiplier applied to the simulation's MinResourceFee
// to derive the submitted transaction fee. For example, 1.25 submits at 25% above
// the simulated minimum, providing headroom during network fee surges. A floor of
// minFeeBuffer (10 000 stroops) is always applied on top of the simulation minimum.
// Non-finite values and values below 1.0 are clamped to 1.0 (no bump).
func WithFeeBumpFactor(factor float64) DeployerOption {
	return func(d *Deployer) {
		if math.IsNaN(factor) || math.IsInf(factor, 0) || factor < 1.0 {
			factor = 1.0
		}
		d.feeBumpFactor = factor
	}
}

// WithTxnTimeBound sets the validity window for every transaction submitted by
// the Deployer. The same duration is used as the transaction's MaxTime and as
// the confirmation poll timeout, keeping them in sync. Durations of zero or
// below are ignored and the default (120s) is kept.
func WithTxnTimeBound(d time.Duration) DeployerOption {
	return func(dep *Deployer) {
		if d > 0 {
			dep.txnTimeBound = d
		}
	}
}

// Deployer handles Soroban contract deployment and initialization.
type Deployer struct {
	rpcClient         stellarRPCClient
	networkPassphrase string
	signer            *keypair.Full
	// accountSequence tracks the current on-chain sequence number.
	accountSequence int64
	// autoRestore controls automatic RestoreFootprint handling for expired
	// persistent ledger entries. True by default.
	autoRestore bool
	// feeBumpFactor is multiplied by the simulation's MinResourceFee to derive
	// the submitted fee. A value of 1.25 means 25% above the simulation minimum.
	// Defaults to 1.25; WithFeeBumpFactor clamps non-finite values and values below 1.0.
	feeBumpFactor float64
	// txnTimeBound is the validity window for every Soroban/classic transaction
	// submitted by the Deployer. waitForTransaction polls for this exact duration,
	// so the poll deadline and the transaction's MaxTime are always in sync.
	// Defaults to defaultTxnTimeBound (120s).
	txnTimeBound time.Duration
}

// NewDeployer creates a new Deployer instance. Options can be passed to
// customise behaviour (e.g. WithAutoRestore(false) to disable automatic
// restoration of expired persistent entries).
func NewDeployer(rpcClient *rpcclient.Client, networkPassphrase string, signer *keypair.Full, opts ...DeployerOption) *Deployer {
	deployer := &Deployer{
		rpcClient:         rpcClient,
		networkPassphrase: networkPassphrase,
		signer:            signer,
		accountSequence:   -1,
		autoRestore:       true,
		feeBumpFactor:     1.25,
		txnTimeBound:      defaultTxnTimeBound,
	}
	for _, opt := range opts {
		opt(deployer)
	}

	return deployer
}

// DeployContract deploys a Soroban contract from a WASM file and returns the contract ID.
// This performs two operations:
// 1. Upload the WASM code (installContractCode)
// 2. Deploy a contract instance (createContract)
func (d *Deployer) DeployContract(ctx context.Context, wasmBytes []byte, salt [32]byte) (string, error) {
	wasmHash, err := d.uploadWASM(ctx, wasmBytes)
	if err != nil {
		return "", fmt.Errorf("failed to upload WASM: %w", err)
	}

	// Create contract instance
	contractID, err := d.createContractInstance(ctx, wasmHash, salt)
	if err != nil {
		return "", fmt.Errorf("failed to create contract instance: %w", err)
	}

	return contractID, nil
}

// uploadWASM uploads WASM code to the network and returns the code hash.
func (d *Deployer) uploadWASM(ctx context.Context, wasmBytes []byte) (xdr.Hash, error) {
	// Get source account
	sourceAccount, err := d.getSourceAccount(ctx)
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to get source account: %w", err)
	}

	// Build upload WASM operation
	uploadOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeUploadContractWasm,
			Wasm: &wasmBytes,
		},
		SourceAccount: d.signer.Address(),
	}

	// Build and submit transaction
	resultMeta, err := d.buildAndSubmitTransaction(ctx, sourceAccount, uploadOp)
	if err != nil {
		return xdr.Hash{}, err
	}

	// Extract WASM hash from result
	wasmHash, err := extractWASMHash(resultMeta)
	if err != nil {
		return xdr.Hash{}, fmt.Errorf("failed to extract WASM hash: %w", err)
	}

	return wasmHash, nil
}

// createContractInstance creates a new contract instance from uploaded WASM code.
func (d *Deployer) createContractInstance(ctx context.Context, wasmHash xdr.Hash, salt [32]byte) (string, error) {
	// Get source account
	sourceAccount, err := d.getSourceAccount(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get source account: %w", err)
	}

	// Get deployer's public key bytes
	pubKeyBytes, err := strkey.Decode(strkey.VersionByteAccountID, d.signer.Address())
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}
	var pubKey256 xdr.Uint256
	copy(pubKey256[:], pubKeyBytes)

	// Build create contract operation
	createOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeCreateContract,
			CreateContract: &xdr.CreateContractArgs{
				ContractIdPreimage: xdr.ContractIdPreimage{
					Type: xdr.ContractIdPreimageTypeContractIdPreimageFromAddress,
					FromAddress: &xdr.ContractIdPreimageFromAddress{
						Address: xdr.ScAddress{
							Type: xdr.ScAddressTypeScAddressTypeAccount,
							AccountId: &xdr.AccountId{
								Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
								Ed25519: &pubKey256,
							},
						},
						Salt: xdr.Uint256(salt),
					},
				},
				Executable: xdr.ContractExecutable{
					Type:     xdr.ContractExecutableTypeContractExecutableWasm,
					WasmHash: &wasmHash,
				},
			},
		},
		SourceAccount: d.signer.Address(),
	}

	// Build and submit transaction
	resultMeta, err := d.buildAndSubmitTransaction(ctx, sourceAccount, createOp)
	if err != nil {
		return "", err
	}

	// Extract contract ID from result
	contractID, err := extractContractID(resultMeta)
	if err != nil {
		return "", fmt.Errorf("failed to extract contract ID: %w", err)
	}

	return contractID, nil
}

// getSourceAccount fetches the current account sequence from the network.
// It always queries the ledger to stay in sync with the on-chain state,
// which is safe because every write path waits for transaction confirmation
// (via waitForTransaction) before returning.
func (d *Deployer) getSourceAccount(ctx context.Context) (*txnbuild.SimpleAccount, error) {
	accountKey := xdr.LedgerKey{
		Type: xdr.LedgerEntryTypeAccount,
		Account: &xdr.LedgerKeyAccount{
			AccountId: xdr.MustAddress(d.signer.Address()),
		},
	}

	keyXDR, err := accountKey.MarshalBinaryBase64()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal account key: %w", err)
	}

	resp, err := d.rpcClient.GetLedgerEntries(ctx, protocolrpc.GetLedgerEntriesRequest{
		Keys: []string{keyXDR},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}

	if len(resp.Entries) == 0 {
		d.accountSequence = 0
	} else {
		entryXDR, ok := getLedgerEntryXDR(resp.Entries[0])
		if !ok || entryXDR == "" {
			d.accountSequence = 0
		} else {
			var entry xdr.LedgerEntryData
			if err := xdr.SafeUnmarshalBase64(entryXDR, &entry); err != nil {
				return nil, fmt.Errorf("failed to unmarshal account entry: %w", err)
			}
			account := entry.MustAccount()
			d.accountSequence = int64(account.SeqNum)
		}
	}

	return &txnbuild.SimpleAccount{
		AccountID: d.signer.Address(),
		Sequence:  d.accountSequence,
	}, nil
}

// buildAndSubmitTransaction builds, signs, and submits a transaction.
func (d *Deployer) buildAndSubmitTransaction(ctx context.Context, sourceAccount *txnbuild.SimpleAccount, op txnbuild.Operation) (*xdr.TransactionMeta, error) {
	// Establish a single deadline shared by the transaction's time-bound and the
	// confirmation poll, so they are always in sync. After a successful auto-restore
	// path, the deadline is refreshed (restore can consume most of the initial window).
	txnDeadline := time.Now().Add(d.txnTimeBound)

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{op},
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, txnDeadline.Unix())},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	txXDR, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction XDR: %w", err)
	}

	simResult, err := d.rpcClient.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{
		Transaction: txXDR,
	})
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	if simResult.Error != "" {
		return nil, fmt.Errorf("simulation error: %s", simResult.Error)
	}

	// If the simulation indicates expired persistent ledger entries, restore
	// them first, then rebuild and re-simulate the original transaction.
	if d.autoRestore && simResult.RestorePreamble != nil {
		if err = d.restoreFootprint(ctx, *simResult.RestorePreamble); err != nil {
			return nil, fmt.Errorf("failed to restore expired ledger entries: %w", err)
		}

		sourceAccount, err = d.getSourceAccount(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get source account after restore: %w", err)
		}

		// Restore + re-fetch can use most of the window started at function entry. Refresh so
		// invoke timebounds, assembleTransaction, and waitForTransaction stay aligned and valid.
		txnDeadline = time.Now().Add(d.txnTimeBound)

		tx, err = txnbuild.NewTransaction(
			txnbuild.TransactionParams{
				SourceAccount:        sourceAccount,
				IncrementSequenceNum: true,
				Operations:           []txnbuild.Operation{op},
				BaseFee:              txnbuild.MinBaseFee,
				Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, txnDeadline.Unix())},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to rebuild transaction after restore: %w", err)
		}

		txXDR, err = tx.Base64()
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction XDR after restore: %w", err)
		}

		simResult, err = d.rpcClient.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{
			Transaction: txXDR,
		})
		if err != nil {
			return nil, fmt.Errorf("simulation failed after restore: %w", err)
		}

		if simResult.Error != "" {
			return nil, fmt.Errorf("simulation error after restore: %s", simResult.Error)
		}
		if simResult.RestorePreamble != nil {
			// The RPC should capture all archived entries in a single RestorePreamble.
			// A second one after a successful restore indicates an RPC inconsistency or
			// an entry that expired in the narrow window between restore confirmation and
			// re-simulation. Return an error so the caller retries from a clean state.
			return nil, errors.New("simulation after restore still requires another restore: unexpected second RestorePreamble")
		}
	}

	assembledTx, err := d.assembleTransaction(ctx, tx, simResult, txnDeadline)
	if err != nil {
		return nil, fmt.Errorf("failed to assemble transaction: %w", err)
	}

	signedTx, err := assembledTx.Sign(d.networkPassphrase, d.signer)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	signedXDR, err := signedTx.Base64()
	if err != nil {
		return nil, fmt.Errorf("failed to get signed transaction XDR: %w", err)
	}

	submitResult, err := d.rpcClient.SendTransaction(ctx, protocolrpc.SendTransactionRequest{
		Transaction: signedXDR,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	switch submitResult.Status {
	case "PENDING", "DUPLICATE": //nolint:goconst
		// Transaction was accepted, continue to wait for confirmation
	case "TRY_AGAIN_LATER": //nolint:goconst
		return nil, errors.New("transaction submission failed: server overloaded, try again later")
	case "ERROR": //nolint:goconst
		if submitResult.ErrorResultXDR != "" {
			return nil, fmt.Errorf("transaction rejected: %v (diagnostics: %v)", submitResult.ErrorResultXDR, submitResult.DiagnosticEventsXDR)
		}

		return nil, errors.New("transaction rejected with status ERROR")
	default:
		return nil, fmt.Errorf("unexpected transaction status: %s", submitResult.Status)
	}

	return d.waitForTransaction(ctx, submitResult.Hash, txnDeadline)
}

// waitForTransaction polls until the transaction is confirmed or the deadline expires.
// The deadline must match the transaction's MaxTime so we stop polling exactly when
// the network will no longer accept the transaction.
func (d *Deployer) waitForTransaction(ctx context.Context, hash string, deadline time.Time) (*xdr.TransactionMeta, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	remaining := time.Until(deadline)
	if remaining <= 0 {
		return nil, fmt.Errorf("transaction deadline already elapsed (hash: %s)", hash)
	}
	timeoutCh := time.After(remaining)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeoutCh:
			return nil, fmt.Errorf("transaction timed out (hash: %s)", hash)
		case <-ticker.C:
			result, err := d.rpcClient.GetTransaction(ctx, protocolrpc.GetTransactionRequest{
				Hash: hash,
			})
			if err != nil {
				continue // transient RPC error, retry
			}

			switch result.Status {
			case "SUCCESS":
				if result.ResultMetaXDR == "" {
					return nil, errors.New("no result meta XDR")
				}
				var meta xdr.TransactionMeta
				if err := xdr.SafeUnmarshalBase64(result.ResultMetaXDR, &meta); err != nil {
					return nil, fmt.Errorf("failed to decode result meta: %w", err)
				}

				return &meta, nil
			case "FAILED":
				return nil, fmt.Errorf("transaction failed (hash: %s, resultXDR: %q, diagnostics: %v)",
					hash, result.ResultXDR, result.DiagnosticEventsXDR)
			case "NOT_FOUND":
				continue // still pending
			}
		}
	}
}

// feeBumpExtra returns stroops to add on top of minFee for a fee bump factor (>= 1).
func feeBumpExtra(minFee int64, factor float64) int64 {
	if minFee <= 0 {
		return 0
	}
	x := float64(minFee) * (factor - 1.0)
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}

	return int64(math.Ceil(x))
}

// restoreFootprint submits a RestoreFootprint transaction using the data provided
// by a simulation's RestorePreamble. Soroban returns this preamble when the
// transaction's read/write footprint references persistent ledger entries whose
// TTL has expired (archived). The restore must succeed before the original
// transaction can be retried.
func (d *Deployer) restoreFootprint(ctx context.Context, preamble protocolrpc.RestorePreamble) error {
	restoreDeadline := time.Now().Add(d.txnTimeBound)

	sourceAccount, err := d.getSourceAccount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get source account for restore: %w", err)
	}

	var sorobanData xdr.SorobanTransactionData
	if err = xdr.SafeUnmarshalBase64(preamble.TransactionDataXDR, &sorobanData); err != nil {
		return fmt.Errorf("failed to decode restore preamble soroban data: %w", err)
	}

	restoreOp := &txnbuild.RestoreFootprint{
		SourceAccount: d.signer.Address(),
		Ext: xdr.TransactionExt{
			V:           1,
			SorobanData: &sorobanData,
		},
	}

	bump := max(feeBumpExtra(preamble.MinResourceFee, d.feeBumpFactor), minFeeBuffer)
	baseFee := preamble.MinResourceFee + bump

	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        sourceAccount,
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{restoreOp},
			BaseFee:              baseFee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, restoreDeadline.Unix())},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build restore transaction: %w", err)
	}

	signedTx, err := tx.Sign(d.networkPassphrase, d.signer)
	if err != nil {
		return fmt.Errorf("failed to sign restore transaction: %w", err)
	}

	signedXDR, err := signedTx.Base64()
	if err != nil {
		return fmt.Errorf("failed to get signed restore transaction XDR: %w", err)
	}

	submitResult, err := d.rpcClient.SendTransaction(ctx, protocolrpc.SendTransactionRequest{
		Transaction: signedXDR,
	})
	if err != nil {
		return fmt.Errorf("failed to submit restore transaction: %w", err)
	}

	switch submitResult.Status {
	case "PENDING", "DUPLICATE":
		// Transaction was accepted
	case "TRY_AGAIN_LATER":
		return errors.New("restore transaction submission failed: server overloaded, try again later")
	case "ERROR":
		if submitResult.ErrorResultXDR != "" {
			return fmt.Errorf("restore transaction rejected: %v (diagnostics: %v)", submitResult.ErrorResultXDR, submitResult.DiagnosticEventsXDR)
		}

		return errors.New("restore transaction rejected with status ERROR")
	default:
		return fmt.Errorf("unexpected restore transaction status: %s", submitResult.Status)
	}

	_, err = d.waitForTransaction(ctx, submitResult.Hash, restoreDeadline)
	if err != nil {
		return fmt.Errorf("restore transaction failed: %w", err)
	}

	return nil
}

// assembleTransaction injects simulation results (Soroban data, auth, fee) into the
// transaction and rebuilds it with the correct fee and the provided deadline as MaxTime.
func (d *Deployer) assembleTransaction(ctx context.Context, tx *txnbuild.Transaction, sim protocolrpc.SimulateTransactionResponse, deadline time.Time) (*txnbuild.Transaction, error) {
	ops := tx.Operations()
	if len(ops) == 0 {
		return tx, nil
	}

	if sim.TransactionDataXDR != "" {
		var sorobanData xdr.SorobanTransactionData
		if err := xdr.SafeUnmarshalBase64(sim.TransactionDataXDR, &sorobanData); err != nil {
			return nil, fmt.Errorf("failed to decode soroban data: %w", err)
		}

		if ihf, ok := ops[0].(*txnbuild.InvokeHostFunction); ok {
			ihf.Ext = xdr.TransactionExt{
				V:           1,
				SorobanData: &sorobanData,
			}

			if len(sim.Results) > 0 && sim.Results[0].AuthXDR != nil && len(*sim.Results[0].AuthXDR) > 0 {
				auth := make([]xdr.SorobanAuthorizationEntry, len(*sim.Results[0].AuthXDR))
				for i, authXDR := range *sim.Results[0].AuthXDR {
					if err := xdr.SafeUnmarshalBase64(authXDR, &auth[i]); err != nil {
						return nil, fmt.Errorf("failed to decode auth: %w", err)
					}
				}
				ihf.Auth = auth
			}
		}
	}

	minFee := sim.MinResourceFee
	if minFee > 0 {
		bump := max(feeBumpExtra(minFee, d.feeBumpFactor), minFeeBuffer)
		newFee := minFee + bump

		sourceAccount, err := d.getSourceAccount(ctx)
		if err != nil {
			return nil, err
		}

		return txnbuild.NewTransaction(
			txnbuild.TransactionParams{
				SourceAccount:        sourceAccount,
				IncrementSequenceNum: true,
				Operations:           ops,
				BaseFee:              newFee,
				Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, deadline.Unix())},
			},
		)
	}

	return tx, nil
}

// extractWASMHash extracts the WASM hash from a transaction result.
func extractWASMHash(meta *xdr.TransactionMeta) (xdr.Hash, error) {
	if meta == nil {
		return xdr.Hash{}, errors.New("nil transaction meta")
	}

	var returnVal *xdr.ScVal

	// Versions below refer to protocol versions (20 and 21+)
	switch meta.V {
	case 4:
		v := meta.MustV4()
		if v.SorobanMeta == nil {
			return xdr.Hash{}, errors.New("no soroban meta")
		}
		returnVal = v.SorobanMeta.ReturnValue
	case 3:
		v := meta.MustV3()
		if v.SorobanMeta == nil {
			return xdr.Hash{}, errors.New("no soroban meta")
		}
		returnVal = &v.SorobanMeta.ReturnValue
	default:
		return xdr.Hash{}, fmt.Errorf("unsupported version: %d", meta.V)
	}

	bytes, ok := returnVal.GetBytes()
	if !ok {
		return xdr.Hash{}, errors.New("return value is not bytes")
	}

	var hash xdr.Hash
	copy(hash[:], bytes)

	return hash, nil
}

// extractContractID extracts the contract ID from a transaction result.
func extractContractID(meta *xdr.TransactionMeta) (string, error) {
	if meta == nil {
		return "", errors.New("nil transaction meta")
	}

	var returnVal *xdr.ScVal

	// Versions below refer to protocol versions (20 and 21+)
	switch meta.V {
	case 4:
		v := meta.MustV4()
		if v.SorobanMeta == nil {
			return "", errors.New("no soroban meta")
		}
		returnVal = v.SorobanMeta.ReturnValue
	case 3:
		v := meta.MustV3()
		if v.SorobanMeta == nil {
			return "", errors.New("no soroban meta")
		}
		returnVal = &v.SorobanMeta.ReturnValue
	default:
		return "", fmt.Errorf("unsupported version: %d", meta.V)
	}

	addr, ok := returnVal.GetAddress()
	if !ok {
		return "", errors.New("return value is not address")
	}

	if addr.Type != xdr.ScAddressTypeScAddressTypeContract {
		return "", errors.New("address is not contract type")
	}

	contractID := addr.MustContractId()

	return strkey.Encode(strkey.VersionByteContract, contractID[:])
}

// GenerateDeterministicSalt generates a deterministic salt for contract deployment.
func GenerateDeterministicSalt(deployerAddress, contractName string) [32]byte {
	saltInput := fmt.Sprintf("%s-%s", deployerAddress, contractName)
	return sha256.Sum256([]byte(saltInput))
}

// getLedgerEntryXDR extracts the XDR from a ledger entry result.
func getLedgerEntryXDR(entry protocolrpc.LedgerEntryResult) (string, bool) {
	if entry.DataXDR != "" {
		return entry.DataXDR, true
	}

	return "", false
}

// // NativeAccountState returns the account native balance (stroops) and sequence number from the ledger.
// // rawAccountKey must be the 32-byte Ed25519 public key (Stellar account id).
// func (d *Deployer) NativeAccountState(ctx context.Context, rawAccountKey []byte) (balance *big.Int, seq uint64, exists bool, err error) {
// 	if len(rawAccountKey) != 32 {
// 		return nil, 0, false, fmt.Errorf("expected 32-byte account key, got %d", len(rawAccountKey))
// 	}
// 	gAddr, err := strkey.Encode(strkey.VersionByteAccountID, rawAccountKey)
// 	if err != nil {
// 		return nil, 0, false, fmt.Errorf("encode account id: %w", err)
// 	}
// 	aid, err := xdr.AddressToAccountId(gAddr)
// 	if err != nil {
// 		return nil, 0, false, fmt.Errorf("account id from address: %w", err)
// 	}
// 	lk := xdr.LedgerKey{
// 		Type: xdr.LedgerEntryTypeAccount,
// 		Account: &xdr.LedgerKeyAccount{
// 			AccountId: aid,
// 		},
// 	}
// 	keyXDR, err := lk.MarshalBinaryBase64()
// 	if err != nil {
// 		return nil, 0, false, fmt.Errorf("marshal account ledger key: %w", err)
// 	}
// 	resp, err := d.rpcClient.GetLedgerEntries(ctx, protocolrpc.GetLedgerEntriesRequest{
// 		Keys: []string{keyXDR},
// 	})
// 	if err != nil {
// 		return nil, 0, false, fmt.Errorf("get ledger entries: %w", err)
// 	}
// 	if len(resp.Entries) == 0 {
// 		return big.NewInt(0), 0, false, nil
// 	}
// 	entryXDR, ok := getLedgerEntryXDR(resp.Entries[0])
// 	if !ok || entryXDR == "" {
// 		return big.NewInt(0), 0, false, nil
// 	}
// 	var entry xdr.LedgerEntryData
// 	if err := xdr.SafeUnmarshalBase64(entryXDR, &entry); err != nil {
// 		return nil, 0, false, fmt.Errorf("unmarshal account entry: %w", err)
// 	}
// 	account := entry.MustAccount()
// 	seqN := int64(account.SeqNum)
// 	if seqN < 0 {
// 		return nil, 0, false, fmt.Errorf("invalid account sequence %d", seqN)
// 	}
//
// 	return big.NewInt(int64(account.Balance)), uint64(seqN), true, nil
// }

// // SubmitClassicOperation builds, signs, and submits a single classic Stellar
// // operation (e.g. ChangeTrust, Payment) via the Soroban RPC.
// // Classic operations cannot be simulated through the Soroban RPC, so this
// // method bypasses simulation and submits the transaction directly.
// func (d *Deployer) SubmitClassicOperation(ctx context.Context, op txnbuild.Operation) error {
// 	src, err := d.getSourceAccount(ctx)
// 	if err != nil {
// 		return fmt.Errorf("load source account: %w", err)
// 	}
//
// 	txnDeadline := time.Now().Add(d.txnTimeBound)
//
// 	tx, err := txnbuild.NewTransaction(
// 		txnbuild.TransactionParams{
// 			SourceAccount:        src,
// 			IncrementSequenceNum: true,
// 			Operations:           []txnbuild.Operation{op},
// 			BaseFee:              txnbuild.MinBaseFee,
// 			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, txnDeadline.Unix())},
// 		},
// 	)
// 	if err != nil {
// 		return fmt.Errorf("build transaction: %w", err)
// 	}
//
// 	signedTx, err := tx.Sign(d.networkPassphrase, d.signer)
// 	if err != nil {
// 		return fmt.Errorf("sign transaction: %w", err)
// 	}
//
// 	signedXDR, err := signedTx.Base64()
// 	if err != nil {
// 		return fmt.Errorf("encode signed transaction: %w", err)
// 	}
//
// 	submitResult, err := d.rpcClient.SendTransaction(ctx, protocolrpc.SendTransactionRequest{
// 		Transaction: signedXDR,
// 	})
// 	if err != nil {
// 		return fmt.Errorf("submit transaction: %w", err)
// 	}
//
// 	switch submitResult.Status {
// 	case "PENDING", "DUPLICATE":
// 	case "TRY_AGAIN_LATER":
// 		return errors.New("transaction submission failed: server overloaded, try again later")
// 	case "ERROR":
// 		if submitResult.ErrorResultXDR != "" {
// 			return fmt.Errorf("transaction rejected: %v (diagnostics: %v)", submitResult.ErrorResultXDR, submitResult.DiagnosticEventsXDR)
// 		}
//
// 		return errors.New("transaction rejected with status ERROR")
// 	default:
// 		return fmt.Errorf("unexpected transaction status: %s", submitResult.Status)
// 	}
//
// 	_, err = d.waitForTransaction(ctx, submitResult.Hash, txnDeadline)
//
// 	return err
// }

// // SendNativePayment submits a payment of stroops native XLM from the deployer's account.
// func (d *Deployer) SendNativePayment(ctx context.Context, destinationStrkey string, stroops int64) error {
// 	if stroops <= 0 {
// 		return fmt.Errorf("payment amount must be positive")
// 	}
// 	if _, err := xdr.AddressToAccountId(destinationStrkey); err != nil {
// 		return fmt.Errorf("invalid destination account: %w", err)
// 	}
// 	src, err := d.getSourceAccount(ctx)
// 	if err != nil {
// 		return fmt.Errorf("load source account: %w", err)
// 	}
// 	payment := &txnbuild.Payment{
// 		Destination: destinationStrkey,
// 		Amount:      fmt.Sprintf("%d", stroops),
// 		Asset:       txnbuild.NativeAsset{},
// 	}
// 	if _, err = d.buildAndSubmitTransaction(ctx, src, payment); err != nil {
// 		return err
// 	}
// 	return nil
// }
//

func (d *Deployer) InvokeContract(ctx context.Context, contractID string, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	// Get source account
	sourceAccount, err := d.getSourceAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get source account: %w", err)
	}

	// Decode contract ID to get raw bytes
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode contract ID: %w", err)
	}

	// Build contract address using XDR marshaling for proper type handling
	contractAddr := buildContractScAddress(contractBytes)
	if contractAddr == nil {
		return nil, errors.New("failed to build contract address")
	}

	// Build invoke operation
	invokeOp := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: *contractAddr,
				FunctionName:    xdr.ScSymbol(functionName),
				Args:            args,
			},
		},
		SourceAccount: d.signer.Address(),
	}

	// Build and submit transaction
	resultMeta, err := d.buildAndSubmitTransaction(ctx, sourceAccount, invokeOp)
	if err != nil {
		return nil, err
	}

	// Extract return value from result
	returnVal, err := extractReturnValue(resultMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to extract return value: %w", err)
	}

	return returnVal, nil
}

func extractReturnValue(meta *xdr.TransactionMeta) (*xdr.ScVal, error) {
	if meta == nil {
		return nil, nil //nolint:nilnil
	}

	// Versions below refer to protocol versions (20 and 21+)
	switch meta.V {
	case 4:
		v := meta.MustV4()
		if v.SorobanMeta == nil {
			return nil, nil //nolint:nilnil
		}

		return v.SorobanMeta.ReturnValue, nil // V4 ReturnValue is already a pointer
	case 3:
		v := meta.MustV3()
		if v.SorobanMeta == nil {
			return nil, nil //nolint:nilnil
		}

		return &v.SorobanMeta.ReturnValue, nil // V3 ReturnValue is a value, need address-of
	default:
		return nil, fmt.Errorf("unsupported transaction meta version: %d", meta.V)
	}
}

func buildContractScAddress(contractIDBytes []byte) *xdr.ScAddress {
	if len(contractIDBytes) != 32 {
		return nil
	}
	xdrBytes := make([]byte, 0, 36)
	xdrBytes = append(xdrBytes, 0, 0, 0, 1)
	xdrBytes = append(xdrBytes, contractIDBytes...)

	var addr xdr.ScAddress
	if err := addr.UnmarshalBinary(xdrBytes); err != nil {
		return nil
	}

	return &addr
}

// SimulateContract simulates a contract invocation without submitting.
func (d *Deployer) SimulateContract(ctx context.Context, contractID string, functionName string, args []xdr.ScVal) (*xdr.ScVal, error) {
	// Decode contract ID to get raw bytes
	contractBytes, err := strkey.Decode(strkey.VersionByteContract, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode contract ID: %w", err)
	}

	// Build contract address using XDR marshaling for proper type handling
	contractAddr := buildContractScAddress(contractBytes)
	if contractAddr == nil {
		return nil, errors.New("failed to build contract address")
	}

	// Build invoke host function
	hostFn := xdr.HostFunction{
		Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
		InvokeContract: &xdr.InvokeContractArgs{
			ContractAddress: *contractAddr,
			FunctionName:    xdr.ScSymbol(functionName),
			Args:            args,
		},
	}

	// Get source account
	sourceAccount, err := d.getSourceAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get source account: %w", err)
	}

	// Build a transaction for simulation
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        sourceAccount,
			IncrementSequenceNum: true,
			Operations: []txnbuild.Operation{
				&txnbuild.InvokeHostFunction{
					HostFunction:  hostFn,
					SourceAccount: d.signer.Address(),
				},
			},
			BaseFee:       txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, time.Now().Add(d.txnTimeBound).Unix())},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction: %w", err)
	}

	// Get transaction envelope XDR
	txXDR, err := tx.Base64()
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction XDR: %w", err)
	}

	// Simulate the transaction
	simResult, err := d.rpcClient.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{
		Transaction: txXDR,
	})
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}

	if simResult.Error != "" {
		return nil, fmt.Errorf("simulation error: %s", simResult.Error)
	}

	// If the simulation indicates expired persistent ledger entries, restore
	// them first, then re-simulate so the read returns fresh data.
	if d.autoRestore && simResult.RestorePreamble != nil {
		if err = d.restoreFootprint(ctx, *simResult.RestorePreamble); err != nil {
			return nil, fmt.Errorf("failed to restore expired ledger entries: %w", err)
		}

		sourceAccount, err = d.getSourceAccount(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get source account after restore: %w", err)
		}

		tx, err = txnbuild.NewTransaction(
			txnbuild.TransactionParams{
				SourceAccount:        sourceAccount,
				IncrementSequenceNum: true,
				Operations: []txnbuild.Operation{
					&txnbuild.InvokeHostFunction{
						HostFunction:  hostFn,
						SourceAccount: d.signer.Address(),
					},
				},
				BaseFee:       txnbuild.MinBaseFee,
				Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewTimebounds(0, time.Now().Add(d.txnTimeBound).Unix())},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to rebuild transaction after restore: %w", err)
		}

		txXDR, err = tx.Base64()
		if err != nil {
			return nil, fmt.Errorf("failed to get transaction XDR after restore: %w", err)
		}

		simResult, err = d.rpcClient.SimulateTransaction(ctx, protocolrpc.SimulateTransactionRequest{
			Transaction: txXDR,
		})
		if err != nil {
			return nil, fmt.Errorf("simulation failed after restore: %w", err)
		}

		if simResult.Error != "" {
			return nil, fmt.Errorf("simulation error after restore: %s", simResult.Error)
		}
		if simResult.RestorePreamble != nil {
			return nil, errors.New("simulation after restore still requires another restore: unexpected second RestorePreamble")
		}
	}

	// Extract result from simulation
	if len(simResult.Results) == 0 {
		return nil, nil //nolint:nilnil
	}

	// The result is returned as a base64-encoded XDR ScVal
	// Try different field names based on SDK version
	result := simResult.Results[0]

	// Try to extract from the result - SDK versions may vary
	// Use the struct's string representation as fallback
	if result.ReturnValueXDR == nil || *result.ReturnValueXDR == "" {
		return nil, nil //nolint:nilnil
	}

	// Access via reflection or try common field patterns
	// The SDK may have the result in different field names
	resultXDR := *result.ReturnValueXDR

	var scVal xdr.ScVal
	if err := xdr.SafeUnmarshalBase64(resultXDR, &scVal); err != nil {
		return nil, fmt.Errorf("failed to decode result: %w", err)
	}

	return &scVal, nil
}
