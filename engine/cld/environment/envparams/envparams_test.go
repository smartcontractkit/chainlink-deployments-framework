package envparams

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

const validDoc = `
domain: example
environment: testnet
catalog:
  grpc: catalog.example.com:443
networks:
  - chain_selector: 16015286601757825753
`

func newFlags(t *testing.T, args ...string) *pflag.FlagSet {
	t.Helper()

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterFlag(fs)
	fs.String(DomainFlagName, "", "")
	fs.String(EnvFlagName, "", "")
	require.NoError(t, fs.Parse(args))

	return fs
}

func envFn(value string) func(string) string {
	return func(key string) string {
		if key == EnvVar {
			return value
		}

		return ""
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		env        string
		wantNil    bool
		wantDomain string
		wantEnv    string
		wantErr    string
	}{
		{
			name:    "nothing supplied falls back to disk",
			wantNil: true,
		},
		{
			name:       "document from variable",
			env:        validDoc,
			wantDomain: "example",
			wantEnv:    "testnet",
		},
		{
			name:       "document from flag",
			args:       []string{"--" + FlagName, validDoc},
			wantDomain: "example",
			wantEnv:    "testnet",
		},
		{
			name:       "flag wins over variable",
			args:       []string{"--" + FlagName, validDoc},
			env:        "domain: other\nenvironment: other\n",
			wantDomain: "example",
			wantEnv:    "testnet",
		},
		{
			name:    "empty flag is an error and does not fall back to variable",
			args:    []string{"--" + FlagName, ""},
			env:     validDoc,
			wantErr: "document is empty",
		},
		{
			name:    "domain flag conflicts with document",
			args:    []string{"--" + DomainFlagName, "example"},
			env:     validDoc,
			wantErr: "--domain cannot be combined with build params",
		},
		{
			name:    "environment flag conflicts with document",
			args:    []string{"--" + EnvFlagName, "testnet"},
			env:     validDoc,
			wantErr: "--environment cannot be combined with build params",
		},
		{
			name:    "explicitly empty environment flag still conflicts",
			args:    []string{"--" + EnvFlagName, ""},
			env:     validDoc,
			wantErr: "--environment cannot be combined with build params",
		},
		{
			name:    "missing domain",
			env:     "environment: testnet\n",
			wantErr: `build params must set "domain"`,
		},
		{
			name:    "missing environment",
			env:     "domain: example\n",
			wantErr: `build params must set "environment"`,
		},
		{
			name:    "unknown key is rejected",
			env:     validDoc + "bogus: true\n",
			wantErr: "failed to decode build params",
		},
		{
			name:    "malformed yaml",
			env:     "domain: [",
			wantErr: "failed to decode build params",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fs := newFlags(t, tt.args...)

			got, err := Resolve(fs, envFn(tt.env))

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			if tt.wantNil {
				require.Nil(t, got)

				return
			}

			require.NotNil(t, got)
			require.Equal(t, tt.wantDomain, got.Domain)
			require.Equal(t, tt.wantEnv, got.Environment)

			// The document's values are written back to the flags and count as set.
			require.Equal(t, tt.wantDomain, fs.Lookup(DomainFlagName).Value.String())
			require.Equal(t, tt.wantEnv, fs.Lookup(EnvFlagName).Value.String())
			require.True(t, fs.Lookup(EnvFlagName).Changed)
		})
	}
}

func TestResolve_ErrorsAreTyped(t *testing.T) {
	t.Parallel()

	_, err := Resolve(newFlags(t, "--"+FlagName, ""), envFn(""))
	require.ErrorIs(t, err, ErrEmptyDocument)

	_, err = Resolve(newFlags(t, "--"+EnvFlagName, "x"), envFn(validDoc))
	require.ErrorIs(t, err, ErrFlagNotAllowed)
}

func TestResolve_UnregisteredFlags(t *testing.T) {
	t.Parallel()

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterFlag(fs)

	got, err := Resolve(fs, envFn(validDoc))
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "testnet", got.Environment)
}

func TestResolve_NilFlags(t *testing.T) {
	t.Parallel()

	got, err := Resolve(nil, envFn(validDoc))
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("empty document", func(t *testing.T) {
		t.Parallel()

		_, err := Parse(nil)
		require.ErrorContains(t, err, "document is empty")
	})

	t.Run("decodes fields", func(t *testing.T) {
		t.Parallel()

		got, err := Parse([]byte(validDoc + "default_rpc_base_url: https://rpc.example.com/\n"))
		require.NoError(t, err)
		require.Equal(t, "catalog.example.com:443", got.Catalog.GRPC)
		require.Len(t, got.Networks, 1)
		require.Equal(t, uint64(16015286601757825753), got.Networks[0].ChainSelector)
		require.Equal(t, "https://rpc.example.com/", got.DefaultRPCBaseURL)
	})
}
