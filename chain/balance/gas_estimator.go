package balance

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rlp"
)

type ChainType string

const (
	ChainTypeL1       ChainType = "L1"
	ChainTypeOPStack  ChainType = "OP_STACK"
	ChainTypeArbitrum ChainType = "ARBITRUM"
	ChainTypeZKRollup ChainType = "ZK_ROLLUP" // zkSync, Scroll, Linea (bundled in EstimateGas)
)

type ChainConfig struct {
	Type              ChainType
	BlockSampleCount  uint64  // 5 for L1, 20-50 for fast L2s
	BaseFeeMultiplier float64 // 2.0 for L1, 1.3 for L2
	MinPriorityFeeWei *big.Int
}

type FeeEstimate struct {
	GasLimit             uint64
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
	L1DataFee            *big.Int
	EstimatedTotalCost   *big.Int // Realistic expected cost in Wei
	MaxTotalCost         *big.Int // Worst-case ceiling cost in Wei
}

// ----------------------

var opGasOracleAddr = common.HexToAddress("0x420000000000000000000000000000000000000F")

func EstimateTransaction(
	ctx context.Context,
	client *ethclient.Client,
	cfg ChainConfig,
	msg ethereum.CallMsg,
	percentile float64, // e.g., 50.0 for standard
) (*FeeEstimate, error) {
	// 1. Estimate execution gas limit
	// Note: Arbitrum and ZK Rollups automatically inflate this limit to cover L1 data/proof costs.
	gasLimit, err := client.EstimateGas(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("estimate gas failed: %w", err)
	}

	// 2. Query eth_feeHistory for EIP-1559 parameters
	history, err := client.FeeHistory(ctx, cfg.BlockSampleCount, nil, []float64{percentile})
	if err != nil || len(history.BaseFee) == 0 {
		// Fallback for non-EIP-1559 chains (e.g., older BNB/alt-L1 networks)
		return fallbackLegacyEstimate(ctx, client, gasLimit)
	}

	// The last element in BaseFee is always the upcoming pending block's base fee
	nextBaseFee := history.BaseFee[len(history.BaseFee)-1]

	// 3. Calculate median priority fee (ignoring empty 0x0 blocks)
	priorityFee := calculateMedianReward(history.Reward, cfg.MinPriorityFeeWei)

	// 4. Apply base fee buffer: MaxFee = (NextBaseFee * Multiplier) + PriorityFee
	bufferedBaseFee := multiplyBigInt(nextBaseFee, cfg.BaseFeeMultiplier)
	maxFeePerGas := new(big.Int).Add(bufferedBaseFee, priorityFee)

	// 5. Add L1 Data Fee for OP Stack chains (Base, Optimism, Blast, Mode, etc.)
	l1DataFee := big.NewInt(0)
	if cfg.Type == ChainTypeOPStack {
		l1DataFee, err = getOPStackL1Fee(ctx, client, msg, gasLimit, maxFeePerGas, priorityFee)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch OP Stack L1 fee: %w", err)
		}
	}

	// 6. Compute total realistic and maximum costs
	gasLimitBig := new(big.Int).SetUint64(gasLimit)
	expectedGasPrice := new(big.Int).Add(nextBaseFee, priorityFee)

	estExecCost := new(big.Int).Mul(gasLimitBig, expectedGasPrice)
	estTotalCost := new(big.Int).Add(estExecCost, l1DataFee)

	maxExecCost := new(big.Int).Mul(gasLimitBig, maxFeePerGas)
	maxTotalCost := new(big.Int).Add(maxExecCost, l1DataFee)

	return &FeeEstimate{
		GasLimit:             gasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: priorityFee,
		L1DataFee:            l1DataFee,
		EstimatedTotalCost:   estTotalCost,
		MaxTotalCost:         maxTotalCost,
	}, nil
}

// -----------------------

func calculateMedianReward(rewards [][]*big.Int, floor *big.Int) *big.Int {
	var validTips []*big.Int
	for _, blockRewards := range rewards {
		if len(blockRewards) > 0 && blockRewards[0].Cmp(big.NewInt(0)) > 0 {
			validTips = append(validTips, blockRewards[0])
		}
	}

	if len(validTips) == 0 {
		if floor != nil {
			return new(big.Int).Set(floor)
		}

		return big.NewInt(0)
	}

	sort.Slice(validTips, func(i, j int) bool {
		return validTips[i].Cmp(validTips[j]) < 0
	})

	median := validTips[len(validTips)/2]
	if floor != nil && median.Cmp(floor) < 0 {
		return new(big.Int).Set(floor)
	}

	return median
}

// getOPStackL1Fee RLP-encodes an unsigned EIP-1559 transaction and calls getL1Fee(bytes)
func getOPStackL1Fee(
	ctx context.Context,
	client *ethclient.Client,
	msg ethereum.CallMsg,
	gasLimit uint64,
	maxFee, maxTip *big.Int,
) (*big.Int, error) {
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	// Build a representative unsigned transaction for accurate byte-size/compression calculation
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
		GasTipCap: maxTip,
		GasFeeCap: maxFee,
		Gas:       gasLimit,
		To:        msg.To,
		Value:     msg.Value,
		Data:      msg.Data,
	})

	rlpBytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return nil, err
	}

	// Selector for getL1Fee(bytes): 0x49948e0e
	selector := crypto.Keccak256([]byte("getL1Fee(bytes)"))[:4]

	// ABI encode the dynamic bytes argument (offset 0x20 + length + padded data)
	paddedLen := ((len(rlpBytes) + 31) / 32) * 32
	calldata := make([]byte, 4+32+32+paddedLen)
	copy(calldata[0:4], selector)
	calldata[35] = 0x20 // Offset to bytes data (32)
	big.NewInt(int64(len(rlpBytes))).FillBytes(calldata[36:68])
	copy(calldata[68:], rlpBytes)

	res, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &opGasOracleAddr,
		Data: calldata,
	}, nil)
	if err != nil {
		return nil, err
	}

	return new(big.Int).SetBytes(res), nil
}

func fallbackLegacyEstimate(ctx context.Context, client *ethclient.Client, gasLimit uint64) (*FeeEstimate, error) {
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	total := new(big.Int).Mul(new(big.Int).SetUint64(gasLimit), gasPrice)

	return &FeeEstimate{
		GasLimit:             gasLimit,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		L1DataFee:            big.NewInt(0),
		EstimatedTotalCost:   total,
		MaxTotalCost:         total,
	}, nil
}

func multiplyBigInt(val *big.Int, multiplier float64) *big.Int {
	// Multiply using 10000 basis points to preserve decimal precision cleanly
	bps := int64(multiplier * 10000)
	res := new(big.Int).Mul(val, big.NewInt(bps))

	return res.Div(res, big.NewInt(10000))
}

// ----------------------

// type BalanceCheckResult struct {
// 	ChainSelector    uint64
// 	ChainName        string
// 	Family           string
// 	Address          string
// 	Symbol           string
// 	Balance          string
// 	BalanceFormatted string
// 	WarnBelow        string
// 	CriticalBelow    string
// 	Status           string // ok | low | critical | unconfigured | error
// 	Message          string
// }
//
// func CheckBalance(ctx context.Context, chains BlockChains /*, thresholds Thresholds*/) []BalanceCheckResult {
// 	for _, chain := range chains.chains {
// 	}
//
//
// 	return nil
// }
