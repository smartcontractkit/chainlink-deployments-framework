package format

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func mustBigInt(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big.Int: " + s)
	}

	return n
}

func TestCommaGroupBigInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *big.Int
		expected string
	}{
		{"nil", nil, "0"},
		{"zero", big.NewInt(0), "0"},
		{"small", big.NewInt(42), "42"},
		{"hundreds", big.NewInt(999), "999"},
		{"thousands", big.NewInt(1000), "1,000"},
		{"millions", big.NewInt(1_000_000), "1,000,000"},
		{"wei", new(big.Int).Mul(big.NewInt(25), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)), "2,500,000,000,000,000,000"},
		{"negative", big.NewInt(-1234567), "-1,234,567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, CommaGroupBigInt(tt.input))
		})
	}
}

func TestFormatTokenAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   *big.Int
		decimals uint8
		expected string
	}{
		{"nil", nil, 18, "0"},
		{"zero", big.NewInt(0), 18, "0"},
		{"6 decimals whole", big.NewInt(1_000_000), 6, "1"},
		{"6 decimals fraction", big.NewInt(1_500_000), 6, "1.5"},
		{"18 decimals large", new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)), 18, "1000"},
		{"18 decimals exact fraction", mustBigInt("2500000000000000000"), 18, "2.5"},
		{
			"exact precision beyond float64",
			mustBigInt("123456789012345678"),
			18,
			"0.123456789012345678",
		},
		{
			"small remainder with leading zeros",
			big.NewInt(1_000_001),
			6,
			"1.000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, FormatTokenAmount(tt.amount, tt.decimals))
		})
	}
}
