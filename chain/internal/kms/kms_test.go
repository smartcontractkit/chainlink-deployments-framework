package kms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ClientConfig_validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    ClientConfig
		wantErr string
	}{
		{
			name: "valid config",
			give: ClientConfig{
				KeyID:     "test-key-id",
				KeyRegion: "us-west-2",
			},
		},
		{
			name: "missing KeyID",
			give: ClientConfig{
				KeyRegion: "us-west-2",
			},
			wantErr: "KMS key ID is required",
		},
		{
			name: "missing KeyRegion",
			give: ClientConfig{
				KeyID: "test-key-id",
			},
			wantErr: "KMS key region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.give.validate()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else if err != nil {
				require.NoError(t, err)
			}
		})
	}
}

func Test_NewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    ClientConfig
		wantErr string
	}{
		{
			name: "valid config using environment variables",
			give: ClientConfig{
				KeyID:     "test-key-id",
				KeyRegion: "us-west-2",
			},
		},
		{
			name: "valid config using profile",
			give: ClientConfig{
				KeyID:      "test-key-id",
				KeyRegion:  "us-west-2",
				AWSProfile: "test-profile",
			},
		},
		{
			name:    "invalid config",
			give:    ClientConfig{},
			wantErr: "invalid KMS config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewClient(tt.give)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}
