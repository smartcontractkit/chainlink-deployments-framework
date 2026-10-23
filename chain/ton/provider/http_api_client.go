package provider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"math/rand/v2"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	tonlib "github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	// httpPollInterval is how often the HTTP client polls for new state while waiting for a
	// transaction to land (the HTTP API has no push mechanism).
	httpPollInterval = 2 * time.Second
	// httpTxWaitFallback bounds SendExternalMessageWaitTransaction when the caller's context
	// has no deadline.
	httpTxWaitFallback = 180 * time.Second
	// httpTxScanBatch is the number of transactions fetched per getTransactions call while scanning.
	httpTxScanBatch = 10
	// httpRequestTimeout bounds a single HTTP API request.
	httpRequestTimeout = 30 * time.Second
	// httpMaxAttempts is the maximum number of attempts per HTTP API call (1 + retries).
	httpMaxAttempts = 5
	// httpBackoffBase is the first retry delay; it doubles on every retry up to httpBackoffMax.
	httpBackoffBase = 500 * time.Millisecond
	// httpBackoffMax caps the exponential backoff between retries.
	httpBackoffMax = 8 * time.Second
	// httpRetryAfterMax caps how long a server provided Retry-After can make us wait.
	httpRetryAfterMax = 30 * time.Second
	// httpMaxResponseBytes caps the size of a single HTTP API response.
	httpMaxResponseBytes = 32 << 20
	// exitCodeOutOfGas is the TVM exit code (-13) the HTTP API returns for get methods on uninitialized accounts.
	exitCodeOutOfGas = -13
	// addrStdBits and addrNoneBits are the sizes of a serialized addr_std (no anycast) and addr_none.
	addrStdBits  = 267
	addrNoneBits = 2
	// tonDecimals is the number of decimals of the TON coin.
	tonDecimals = 9
)

// httpAPIClient implements tonlib.APIClientWrapped on top of the TON HTTP API v2
// (toncenter-compatible: /api/v2/getMasterchainInfo, /runGetMethod, /sendBoc, ...).
//
// Only the methods the MCMS flows (inspect, set-root, execute, timelock, trace confirmation)
// and the tonutils-go wallet rely on are implemented. All other methods fall through to the
// embedded offline client and return an error.
//
// Differences from the liteserver client:
//   - State reads (getters, accounts) are always served at the latest state; the block argument
//     and WaitForBlock are ignored.
//   - There are no Merkle proofs: results are trusted as returned by the HTTP API provider.
type httpAPIClient struct {
	tonlib.APIClientWrapped // offline fallback for unimplemented methods

	baseURL     string // API v2 root, without trailing slash
	apiKey      string
	http        *http.Client
	backoffBase time.Duration
}

var _ tonlib.APIClientWrapped = (*httpAPIClient)(nil)

// newHTTPAPIClient creates a client for a TON HTTP API v2 endpoint. apiURL is the API v2 root
// exactly as served by the provider, i.e. methods are called as <apiURL>/getMasterchainInfo,
// <apiURL>/runGetMethod, ... (for toncenter that is https://toncenter.com/api/v2).
func newHTTPAPIClient(apiURL, apiKey string) *httpAPIClient {
	return &httpAPIClient{
		APIClientWrapped: tonlib.NewAPIClient(liteclient.NewOfflineClient(), tonlib.ProofCheckPolicyUnsafe),
		baseURL:          strings.TrimRight(strings.TrimSpace(apiURL), "/"),
		apiKey:           apiKey,
		http:             &http.Client{Timeout: httpRequestTimeout},
		backoffBase:      httpBackoffBase,
	}
}

// v2Envelope is the response envelope of every v2 method.
type v2Envelope[T any] struct {
	Ok     bool   `json:"ok"`
	Result T      `json:"result"`
	Error  string `json:"error"`
	Code   *int   `json:"code"`
}

// httpStatusError is a non-2xx HTTP API response.
type httpStatusError struct {
	name       string
	status     int
	msg        string
	retryAfter time.Duration
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("%s: API error (status %d): %s", e.name, e.status, e.msg)
}

// isRetryableStatus reports whether a response status is transient: rate limited or a server error.
func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

// parseRetryAfter parses a Retry-After header given in seconds. HTTP dates are not supported.
func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs <= 0 {
		return 0
	}

	return min(time.Duration(secs)*time.Second, httpRetryAfterMax)
}

// retryDelay returns the wait before retry number attempt (0-based): exponential backoff from
// c.backoffBase capped at httpBackoffMax, with jitter in [d/2, d). A server provided Retry-After
// takes precedence when it is longer.
func (c *httpAPIClient) retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	d := min(c.backoffBase<<attempt, httpBackoffMax)
	if half := d / 2; half > 0 {
		d = half + rand.N(half) //nolint:gosec // jitter, not security sensitive
	}

	return max(d, retryAfter)
}

// retryPolicy selects which failures v2Do retries.
type retryPolicy int

const (
	// retryTransient retries rate limiting (429), server errors (5xx) and transport errors.
	retryTransient retryPolicy = iota
	// retryRateLimitOnly retries only 429, i.e. requests that were definitely not processed. Used for
	// sendBoc: after a 5xx or a transport error the message may already have been broadcast, and a
	// retry would then fail with a misleading "not accepted" (seqno already used) error.
	retryRateLimitOnly
)

// v2Do calls an HTTP API v2 method, retrying failures allowed by policy with exponential backoff, up
// to httpMaxAttempts attempts.
func v2Do[T any](
	ctx context.Context, c *httpAPIClient, policy retryPolicy, method, name string, q url.Values, body any,
) (*T, error) {
	u := c.baseURL + "/" + name
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}

	var lastErr error
	for attempt := range httpMaxAttempts {
		if attempt > 0 {
			var retryAfter time.Duration
			var se *httpStatusError
			if errors.As(lastErr, &se) {
				retryAfter = se.retryAfter
			}
			if err := sleepCtx(ctx, c.retryDelay(attempt-1, retryAfter)); err != nil {
				return nil, fmt.Errorf("%w (last error: %w)", err, lastErr)
			}
		}

		res, retryable, err := v2DoOnce[T](ctx, c, method, name, u, payload)
		if err == nil {
			return res, nil
		}
		lastErr = err
		if policy == retryRateLimitOnly {
			var se *httpStatusError
			retryable = errors.As(err, &se) && se.status == http.StatusTooManyRequests
		}
		if !retryable || ctx.Err() != nil {
			break
		}
	}

	return nil, lastErr
}

// v2DoOnce performs a single request. retryable reports whether the failure is transient.
func v2DoOnce[T any](
	ctx context.Context, c *httpAPIClient, method, name, u string, payload []byte,
) (res *T, retryable bool, err error) {
	var reqBody io.Reader
	if payload != nil {
		reqBody = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, false, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		// do not leak the URL: it may carry an access token
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}

		return nil, true, fmt.Errorf("%s %s: %w", method, name, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, httpMaxResponseBytes))
	if err != nil {
		return nil, true, fmt.Errorf("%s: failed to read response: %w", name, err)
	}

	var env v2Envelope[T]
	jerr := json.Unmarshal(raw, &env)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := env.Error
		if jerr != nil || msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}

		return nil, isRetryableStatus(resp.StatusCode), &httpStatusError{
			name:       name,
			status:     resp.StatusCode,
			msg:        msg,
			retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}
	if jerr != nil {
		return nil, false, fmt.Errorf("%s: unexpected response (status %d): %w", name, resp.StatusCode, jerr)
	}
	if !env.Ok {
		return nil, false, fmt.Errorf("%s: API error (status %d): %s", name, resp.StatusCode, env.Error)
	}

	return &env.Result, false, nil
}

func v2Get[T any](ctx context.Context, c *httpAPIClient, name string, q url.Values) (*T, error) {
	return v2Do[T](ctx, c, retryTransient, http.MethodGet, name, q, nil)
}

func v2Post[T any](ctx context.Context, c *httpAPIClient, name string, body any) (*T, error) {
	return v2Do[T](ctx, c, retryTransient, http.MethodPost, name, nil, body)
}

// blockIDV2 is a block identifier as returned by the v2 API.
type blockIDV2 struct {
	Workchain int32  `json:"workchain"`
	Shard     int64  `json:"shard,string"`
	Seqno     uint64 `json:"seqno"`
	RootHash  []byte `json:"root_hash"`
	FileHash  []byte `json:"file_hash"`
}

// transactionIDV2 is a transaction identifier as returned by the v2 API.
type transactionIDV2 struct {
	LT   uint64 `json:"lt,string"`
	Hash []byte `json:"hash"`
}

// isHTTPAPIURL reports whether the URL selects the HTTP API transport.
func isHTTPAPIURL(u string) bool {
	return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://")
}

// --- no-op / pass-through configuration methods ---

func (c *httpAPIClient) WaitForBlock(uint32) tonlib.APIClientWrapped                 { return c }
func (c *httpAPIClient) WithRetryTimeout(int, time.Duration) tonlib.APIClientWrapped { return c }
func (c *httpAPIClient) WithRetry(...int) tonlib.APIClientWrapped                    { return c }
func (c *httpAPIClient) WithTimeout(time.Duration) tonlib.APIClientWrapped           { return c }
func (c *httpAPIClient) WithLSInfoInErrors() tonlib.APIClientWrapped                 { return c }
func (c *httpAPIClient) SetTrustedBlock(*tonlib.BlockIDExt)                          {}

// --- blocks ---

func toBlockIDExt(b blockIDV2) *tonlib.BlockIDExt {
	return &tonlib.BlockIDExt{
		Workchain: b.Workchain,
		Shard:     b.Shard,
		SeqNo:     uint32(b.Seqno), //nolint:gosec // masterchain seqno fits in uint32
		RootHash:  b.RootHash,
		FileHash:  b.FileHash,
	}
}

// GetMasterchainInfo -> GET /getMasterchainInfo
func (c *httpAPIClient) GetMasterchainInfo(ctx context.Context) (*tonlib.BlockIDExt, error) {
	res, err := v2Get[struct {
		Last blockIDV2 `json:"last"`
	}](ctx, c, "getMasterchainInfo", nil)
	if err != nil {
		return nil, err
	}

	return toBlockIDExt(res.Last), nil
}

// CurrentMasterchainInfo -> GET /getMasterchainInfo
func (c *httpAPIClient) CurrentMasterchainInfo(ctx context.Context) (*tonlib.BlockIDExt, error) {
	return c.GetMasterchainInfo(ctx)
}

// GetBlockHeader -> GET /getBlockHeader. Only SeqNo and GenUtime are populated.
func (c *httpAPIClient) GetBlockHeader(ctx context.Context, block *tonlib.BlockIDExt) (*tlb.BlockHeader, error) {
	res, err := v2Get[struct {
		ID       blockIDV2 `json:"id"`
		GenUtime uint32    `json:"gen_utime"`
	}](ctx, c, "getBlockHeader", url.Values{
		"workchain": {strconv.FormatInt(int64(block.Workchain), 10)},
		"shard":     {strconv.FormatInt(block.Shard, 10)},
		"seqno":     {strconv.FormatUint(uint64(block.SeqNo), 10)},
		// some providers refuse the lookup without the hashes ("mc block is older than block's masterchain ref")
		"root_hash": {base64.StdEncoding.EncodeToString(block.RootHash)},
		"file_hash": {base64.StdEncoding.EncodeToString(block.FileHash)},
	})
	if err != nil {
		return nil, err
	}

	return &tlb.BlockHeader{
		SeqNo:    uint32(res.ID.Seqno), //nolint:gosec // masterchain seqno fits in uint32
		GenUtime: res.GenUtime,
	}, nil
}

// GetBlockData is backed by GET /getBlockHeader, so only BlockInfo.SeqNo and BlockInfo.GenUtime
// are populated (MCMS only reads GenUtime).
func (c *httpAPIClient) GetBlockData(ctx context.Context, block *tonlib.BlockIDExt) (*tlb.Block, error) {
	hdr, err := c.GetBlockHeader(ctx, block)
	if err != nil {
		return nil, err
	}

	return &tlb.Block{BlockInfo: *hdr}, nil
}

// --- get methods ---

type runGetMethodRequest struct {
	Address string     `json:"address"`
	Method  string     `json:"method"`
	Stack   [][]string `json:"stack"`
}

type runGetMethodResponse struct {
	GasUsed  uint64              `json:"gas_used"`
	Stack    [][]json.RawMessage `json:"stack"`
	ExitCode int32               `json:"exit_code"`
}

// RunGetMethod -> POST /runGetMethod. The block argument is ignored (latest state is used).
func (c *httpAPIClient) RunGetMethod(
	ctx context.Context, _ *tonlib.BlockIDExt, addr *address.Address, method string, params ...any,
) (*tonlib.ExecutionResult, error) {
	stack := make([][]string, 0, len(params))
	for i, p := range params {
		entry, err := encodeStackParam(p)
		if err != nil {
			return nil, fmt.Errorf("runGetMethod %q: param %d: %w", method, i, err)
		}
		stack = append(stack, entry)
	}

	res, err := v2Post[runGetMethodResponse](ctx, c, "runGetMethod", runGetMethodRequest{
		Address: addr.String(),
		Method:  method,
		Stack:   stack,
	})
	if err != nil {
		return nil, fmt.Errorf("runGetMethod %q: %w", method, err)
	}

	if res.ExitCode == exitCodeOutOfGas {
		// The HTTP API reports a get method on an uninitialized account as exit code -13, whereas the
		// liteserver client yields ErrCodeContractNotInitialized, which the wallet relies on (seqno = 0).
		return nil, tonlib.ContractExecError{Code: tonlib.ErrCodeContractNotInitialized}
	}
	if res.ExitCode != 0 && res.ExitCode != 1 {
		return nil, tonlib.ContractExecError{Code: res.ExitCode}
	}

	values := make([]any, 0, len(res.Stack))
	for i, raw := range res.Stack {
		v, err := decodeStackEntry(raw)
		if err != nil {
			return nil, fmt.Errorf("runGetMethod %q: result stack[%d]: %w", method, i, err)
		}
		values = append(values, v)
	}

	return tonlib.NewExecutionResult(values), nil
}

// RunGetMethodByID is not supported over the HTTP API client.
func (c *httpAPIClient) RunGetMethodByID(
	context.Context, *tonlib.BlockIDExt, *address.Address, uint64, ...any,
) (*tonlib.ExecutionResult, error) {
	return nil, errors.New("RunGetMethodByID is not supported by the TON HTTP API client")
}

func bigToHex(v *big.Int) string {
	if v.Sign() < 0 {
		return "-0x" + new(big.Int).Neg(v).Text(16)
	}

	return "0x" + v.Text(16)
}

func encodeStackParam(p any) ([]string, error) {
	switch v := p.(type) {
	case *big.Int:
		if v == nil {
			return nil, errors.New("nil big.Int")
		}

		return []string{"num", bigToHex(v)}, nil
	case bool:
		if v {
			return []string{"num", "-0x1"}, nil // TVM true is -1
		}

		return []string{"num", "0x0"}, nil
	case *address.Address:
		return []string{"tvm.Slice", base64.StdEncoding.EncodeToString(cell.BeginCell().MustStoreAddr(v).EndCell().ToBOC())}, nil
	case *cell.Slice:
		return []string{"tvm.Slice", base64.StdEncoding.EncodeToString(v.MustToCell().ToBOC())}, nil
	case *cell.Cell:
		return []string{"tvm.Cell", base64.StdEncoding.EncodeToString(v.ToBOC())}, nil
	}

	rv := reflect.ValueOf(p)
	switch rv.Kind() { //nolint:exhaustive // only integer kinds are supported
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []string{"num", bigToHex(big.NewInt(rv.Int()))}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return []string{"num", bigToHex(new(big.Int).SetUint64(rv.Uint()))}, nil
	default:
		return nil, fmt.Errorf("unsupported stack param type %T", p)
	}
}

// decodeStackEntry converts a v2 stack entry ([type, value]) into the value types used by
// tonlib.ExecutionResult: *big.Int, *cell.Cell, *cell.Slice or nil.
func decodeStackEntry(raw []json.RawMessage) (any, error) {
	if len(raw) != 2 { //nolint:mnd // [type, value]
		return nil, fmt.Errorf("malformed stack entry of length %d", len(raw))
	}

	var typ string
	if err := json.Unmarshal(raw[0], &typ); err != nil {
		return nil, fmt.Errorf("malformed stack entry type: %w", err)
	}

	switch typ {
	case "num":
		var s string
		if err := json.Unmarshal(raw[1], &s); err != nil {
			return nil, fmt.Errorf("malformed num value: %w", err)
		}

		return parseHexInt(s)
	case "cell", "slice":
		boc, err := bocFromStackValue(raw[1])
		if err != nil {
			return nil, fmt.Errorf("malformed %s value: %w", typ, err)
		}
		c, err := cell.FromBOC(boc)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s BOC: %w", typ, err)
		}
		// The HTTP API serializes TVM slices as "cell" entries, so a slice cannot be told apart from a cell by
		// type. MCMS getters return addresses as slices and dictionaries/data as cells; an address-only,
		// ref-less cell is therefore decoded as a slice. A dictionary root can never look like that.
		if typ == "slice" || isAddressCell(c) {
			return c.BeginParse()
		}

		return c, nil
	case "list":
		// The HTTP API (tonlib) encodes a TVM null as an empty list: ["list", {"elements": []}].
		var l struct {
			Elements []json.RawMessage `json:"elements"`
		}
		if err := json.Unmarshal(raw[1], &l); err != nil {
			return nil, fmt.Errorf("malformed list value: %w", err)
		}
		if len(l.Elements) != 0 {
			return nil, fmt.Errorf("unsupported non-empty list stack entry (%d elements)", len(l.Elements))
		}

		return nil, nil //nolint:nilnil // TVM null
	case "null", "unsupported":
		return nil, nil //nolint:nilnil // a nil stack value is a valid TVM null
	default:
		return nil, fmt.Errorf("unsupported stack entry type %q", typ)
	}
}

// isAddressCell reports whether the cell holds exactly one MsgAddress (addr_std, 267 bits, or addr_none,
// 2 bits) and nothing else.
func isAddressCell(c *cell.Cell) bool {
	if c.RefsNum() != 0 || (c.BitsSize() != addrStdBits && c.BitsSize() != addrNoneBits) {
		return false
	}

	s, err := c.BeginParse()
	if err != nil {
		return false
	}
	if _, err = s.LoadAddr(); err != nil {
		return false
	}

	return s.BitsLeft() == 0
}

func parseHexInt(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "0x")

	v, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid number %q", s)
	}
	if neg {
		v.Neg(v)
	}

	return v, nil
}

// bocFromStackValue accepts both {"bytes": "<base64 boc>", ...} and a plain base64 string.
func bocFromStackValue(raw json.RawMessage) ([]byte, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return base64.StdEncoding.DecodeString(s)
	}

	var obj struct {
		Bytes string `json:"bytes"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(obj.Bytes)
}

// --- accounts ---

type addressInformationResponse struct {
	Balance           string          `json:"balance"`
	Code              string          `json:"code"`
	Data              string          `json:"data"`
	LastTransactionID transactionIDV2 `json:"last_transaction_id"`
	State             string          `json:"state"` // "active", "uninitialized", "frozen"
}

func decodeOptionalCell(b64 string) (*cell.Cell, error) {
	if b64 == "" {
		return nil, nil //nolint:nilnil // absent cell
	}
	boc, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}

	return cell.FromBOC(boc)
}

// GetAccount -> GET /getAddressInformation. The block argument is ignored (latest state is used).
// Only the fields consumed by the wallet and trace tracking are populated: IsActive, State
// (status, balance, state init), Code, Data, LastTxLT and LastTxHash.
func (c *httpAPIClient) GetAccount(ctx context.Context, _ *tonlib.BlockIDExt, addr *address.Address) (*tlb.Account, error) {
	res, err := v2Get[addressInformationResponse](ctx, c, "getAddressInformation",
		url.Values{"address": {addr.String()}})
	if err != nil {
		return nil, err
	}

	acc := &tlb.Account{
		LastTxLT:   res.LastTransactionID.LT,
		LastTxHash: res.LastTransactionID.Hash,
	}
	if res.State != "active" {
		return acc, nil
	}

	nano, ok := new(big.Int).SetString(res.Balance, 10) //nolint:mnd // decimal
	if !ok {
		return nil, fmt.Errorf("getAddressInformation: invalid balance %q", res.Balance)
	}
	balance, err := tlb.FromNano(nano, tonDecimals)
	if err != nil {
		return nil, fmt.Errorf("getAddressInformation: invalid balance: %w", err)
	}
	code, err := decodeOptionalCell(res.Code)
	if err != nil {
		return nil, fmt.Errorf("getAddressInformation: invalid code: %w", err)
	}
	data, err := decodeOptionalCell(res.Data)
	if err != nil {
		return nil, fmt.Errorf("getAddressInformation: invalid data: %w", err)
	}

	acc.IsActive = true
	acc.Code = code
	acc.Data = data
	acc.State = &tlb.AccountState{
		IsValid: true,
		Address: addr,
		AccountStorage: tlb.AccountStorage{
			Status:            tlb.AccountStatusActive,
			LastTransactionLT: res.LastTransactionID.LT,
			Balance:           balance,
			StateInit:         &tlb.StateInit{Code: code, Data: data},
		},
	}

	return acc, nil
}

// --- transactions ---

type rawTransactionV2 struct {
	TransactionID transactionIDV2 `json:"transaction_id"`
	Data          []byte          `json:"data"` // base64 BOC of the transaction cell
}

// ListTransactions -> GET /getTransactions. Like the liteserver client, it returns transactions
// before (and including) the given lt/hash, oldest first. With lt == 0 it starts at the latest.
func (c *httpAPIClient) ListTransactions(
	ctx context.Context, addr *address.Address, num uint32, lt uint64, txHash []byte,
) ([]*tlb.Transaction, error) {
	q := url.Values{
		"address":  {addr.String()},
		"limit":    {strconv.FormatUint(uint64(num), 10)},
		"archival": {"true"},
	}
	if lt != 0 {
		q.Set("lt", strconv.FormatUint(lt, 10))
		q.Set("hash", hex.EncodeToString(txHash))
	}

	res, err := v2Get[[]rawTransactionV2](ctx, c, "getTransactions", q)
	if err != nil {
		return nil, err
	}
	if len(*res) == 0 {
		return nil, tonlib.ErrNoTransactionsWereFound
	}

	// The API returns newest first.
	out := make([]*tlb.Transaction, len(*res))
	for i, raw := range *res {
		root, err := cell.FromBOC(raw.Data)
		if err != nil {
			return nil, fmt.Errorf("getTransactions: failed to parse transaction BOC: %w", err)
		}

		var tx tlb.Transaction
		if err = tlb.Parse(&tx, root); err != nil {
			return nil, fmt.Errorf("getTransactions: failed to load transaction: %w", err)
		}
		tx.Hash = root.Hash()
		if !bytes.Equal(tx.Hash, raw.TransactionID.Hash) {
			return nil, errors.New("getTransactions: transaction hash does not match transaction_id")
		}

		out[len(*res)-1-i] = &tx
	}

	return out, nil
}

// SendExternalMessage -> POST /sendBoc
func (c *httpAPIClient) SendExternalMessage(ctx context.Context, msg *tlb.ExternalMessage) error {
	root, err := tlb.ToCell(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize external message: %w", err)
	}

	body := map[string][]byte{"boc": root.ToBOCWithFlags(false)}
	if _, err = v2Do[json.RawMessage](ctx, c, retryRateLimitOnly, http.MethodPost, "sendBoc", nil, body); err != nil {
		return err
	}

	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// SendExternalMessageWaitTransaction sends the message and then polls the destination account
// (getAddressInformation / getTransactions) until the transaction carrying the message is found.
func (c *httpAPIClient) SendExternalMessageWaitTransaction(
	ctx context.Context, ext *tlb.ExternalMessage,
) (*tlb.Transaction, *tonlib.BlockIDExt, []byte, error) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, httpTxWaitFallback)
		defer cancel()
	}
	till, _ := ctx.Deadline()

	acc, err := c.GetAccount(ctx, nil, ext.DstAddr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get account state: %w", err)
	}

	inMsgHash := ext.Body.Hash()

	if err = c.SendExternalMessage(ctx, ext); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to send message: %w", err)
	}

	for time.Now().Before(till) {
		if err = sleepCtx(ctx, httpPollInterval); err != nil {
			break
		}

		accNew, err := c.GetAccount(ctx, nil, ext.DstAddr)
		if err != nil || accNew.LastTxLT == acc.LastTxLT {
			continue
		}

		tx, err := c.findExternalInTx(ctx, till, ext, acc, accNew)
		if err != nil {
			return nil, nil, nil, err
		}
		if tx == nil {
			acc = accNew

			continue
		}

		block, err := c.GetMasterchainInfo(ctx)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get block: %w", err)
		}

		return tx, block, inMsgHash, nil
	}

	return nil, nil, nil, tonlib.ErrTxWasNotConfirmed
}

// findExternalInTx scans the account's transactions from accNew back to acc looking for the one
// whose external-in message matches ext. It returns (nil, nil) if it is not among them.
func (c *httpAPIClient) findExternalInTx(
	ctx context.Context, till time.Time, ext *tlb.ExternalMessage, acc, accNew *tlb.Account,
) (*tlb.Transaction, error) {
	lastLT, lastHash := accNew.LastTxLT, accNew.LastTxHash

	for time.Now().Before(till) {
		txList, err := c.ListTransactions(ctx, ext.DstAddr, httpTxScanBatch, lastLT, lastHash)
		if err != nil {
			if sleepErr := sleepCtx(ctx, httpPollInterval); sleepErr != nil {
				return nil, nil //nolint:nilnil // context done, caller times out
			}

			continue
		}

		sawLastTx := false
		for i, tx := range txList {
			if i == 0 {
				// oldest tx of this batch: continue scanning from its predecessor if needed
				lastLT, lastHash = tx.PrevTxLT, tx.PrevTxHash
			}

			if !sawLastTx && tx.PrevTxLT == acc.LastTxLT && bytes.Equal(tx.PrevTxHash, acc.LastTxHash) {
				sawLastTx = true
			}

			if tx.IO.In == nil || tx.IO.In.MsgType != tlb.MsgTypeExternalIn {
				continue
			}
			extIn := tx.IO.In.AsExternalIn()
			if ext.StateInit != nil {
				if extIn.StateInit == nil ||
					ext.StateInit.Data.HashKey() != extIn.StateInit.Data.HashKey() ||
					ext.StateInit.Code.HashKey() != extIn.StateInit.Code.HashKey() {
					continue
				}
			}
			if extIn.Body.HashKey() != ext.Body.HashKey() {
				continue
			}

			return tx, nil
		}

		if sawLastTx || lastLT == 0 {
			return nil, nil //nolint:nilnil // not found in the new transactions
		}
	}

	return nil, nil //nolint:nilnil // deadline reached
}

// SubscribeOnTransactions polls the account and emits, oldest first, every transaction newer than
// lastProcessedLT. It closes channel when workerCtx is done.
func (c *httpAPIClient) SubscribeOnTransactions(
	workerCtx context.Context, addr *address.Address, lastProcessedLT uint64, channel chan<- *tlb.Transaction,
) {
	defer close(channel)

	const pollWait = 3 * time.Second
	const reqTimeout = 10 * time.Second

	wait := time.Duration(0)
	for {
		if err := sleepCtx(workerCtx, wait); err != nil {
			return
		}
		wait = pollWait

		ctx, cancel := context.WithTimeout(workerCtx, reqTimeout)
		acc, err := c.GetAccount(ctx, nil, addr)
		cancel()
		if err != nil || !acc.IsActive || acc.LastTxLT == 0 || acc.LastTxLT == lastProcessedLT {
			continue
		}

		// Collect transactions newest -> oldest until we reach the already processed ones.
		var transactions []*tlb.Transaction
		lastHash, lastLT := acc.LastTxHash, acc.LastTxLT

	list:
		for lastLT != 0 {
			ctx, cancel = context.WithTimeout(workerCtx, reqTimeout)
			res, err := c.ListTransactions(ctx, addr, httpTxScanBatch, lastLT, lastHash)
			cancel()
			if err != nil {
				if errors.Is(err, tonlib.ErrNoTransactionsWereFound) && len(transactions) > 0 {
					break
				}
				if sleepCtx(workerCtx, pollWait) != nil {
					return
				}

				continue
			}

			// res is oldest first; walk it newest first
			for i := len(res) - 1; i >= 0; i-- {
				if res[i].LT <= lastProcessedLT {
					break list
				}
				transactions = append(transactions, res[i])
			}

			lastLT, lastHash = res[0].PrevTxLT, res[0].PrevTxHash
		}

		if len(transactions) == 0 {
			continue
		}

		lastProcessedLT = transactions[0].LT
		for i := len(transactions) - 1; i >= 0; i-- { // oldest first
			select {
			case channel <- transactions[i]:
			case <-workerCtx.Done():
				return
			}
		}

		wait = 0
	}
}
