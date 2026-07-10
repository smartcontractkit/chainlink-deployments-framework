package chains

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNonceTooLowError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns false for nil error",
			err:  nil,
			want: false,
		},
		{
			name: "matches nonce too low message",
			err:  errors.New("nonce too low: address 0xabc, tx: 1 state: 2"),
			want: true,
		},
		{
			name: "matches case-insensitively",
			err:  errors.New("Error Nonce Too Low"),
			want: true,
		},
		{
			name: "matches wrapped error",
			err:  fmt.Errorf("send failed: %w", errors.New("nonce too low")),
			want: true,
		},
		{
			name: "does not match unrelated error",
			err:  errors.New("replacement transaction underpriced"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isNonceTooLowError(tt.err))
		})
	}
}

func TestIsNoContractCodeError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "returns false for nil error",
			err:  nil,
			want: false,
		},
		{
			name: "matches no contract code message",
			err:  errors.New("no contract code at given address"),
			want: true,
		},
		{
			name: "matches wrapped error",
			err:  fmt.Errorf("call failed: %w", errors.New("no contract code")),
			want: true,
		},
		{
			name: "does not match unrelated error",
			err:  errors.New("execution reverted"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, isNoContractCodeError(tt.err))
		})
	}
}
