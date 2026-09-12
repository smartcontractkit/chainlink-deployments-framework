// Package verification provides pre-hooks that enforce contract verification
// before changeset execution. Use RequireVerified to block changesets when
// referenced contracts must be verified on block explorers first.
package verification

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/verification/evm"
)

const hookTimeout = 60 * time.Second

// RequireVerifiedOption configures RequireVerified behavior.
type RequireVerifiedOption func(*requireVerifiedOpts)

type requireVerifiedOpts struct {
	httpClient *http.Client
}

// WithHTTPClient sets the HTTP client for block explorer API calls. Use for testing.
func WithHTTPClient(c *http.Client) RequireVerifiedOption {
	return func(o *requireVerifiedOpts) {
		o.httpClient = c
	}
}

// RefsProvider returns the address refs to check for a given changeset.
// Return nil or empty slice to skip verification for that changeset.
type RefsProvider func(params changeset.PreHookParams) ([]datastore.AddressRef, error)

// RefsForChangeset returns a RefsProvider that looks up refs by changeset key.
// Use this when each changeset has a fixed set of contracts to verify.
func RefsForChangeset(m map[string][]datastore.AddressRef) RefsProvider {
	return func(params changeset.PreHookParams) ([]datastore.AddressRef, error) {
		return m[params.ChangesetKey], nil
	}
}

// RequireVerified returns a PreHook that blocks changeset execution when any
// of the provided refs are not verified on block explorers. Uses evm.CheckVerified
// under the hood. Skips when refsProvider returns no refs.
//
// Requires domain (for loading network config), a refsProvider (to get refs per
// changeset), and a ContractInputsProvider (for contract metadata). Uses Abort
// policy with a 60s timeout.
func RequireVerified(
	domain fdomain.Domain,
	refsProvider RefsProvider,
	contractInputsProvider evm.ContractInputsProvider,
	opts ...RequireVerifiedOption,
) changeset.PreHook {
	var o requireVerifiedOpts
	for _, opt := range opts {
		opt(&o)
	}

	return changeset.PreHook{
		HookDefinition: changeset.HookDefinition{
			Name:          "require-verified",
			FailurePolicy: changeset.Abort,
			Timeout:       hookTimeout,
		},
		Func: func(ctx context.Context, params changeset.PreHookParams) error {
			refs, err := refsProvider(params)
			if err != nil {
				return fmt.Errorf("require-verified: get refs: %w", err)
			}
			if len(refs) == 0 {
				return nil
			}

			networkCfg, err := config.LoadNetworks(params.Env.Name, domain, params.Env.Logger)
			if err != nil {
				return fmt.Errorf("require-verified: load networks: %w", err)
			}

			checkCfg := evm.CheckConfig{
				ContractInputsProvider: contractInputsProvider,
				NetworkConfig:          networkCfg,
				Logger:                 params.Env.Logger,
			}
			if o.httpClient != nil {
				checkCfg.HTTPClient = o.httpClient
			}
			unverified, err := evm.CheckVerified(ctx, refs, checkCfg)
			if err != nil {
				return fmt.Errorf("require-verified: %w", err)
			}
			if len(unverified) > 0 {
				return fmt.Errorf("require-verified: %d contract(s) not verified: %v", len(unverified), unverified)
			}

			return nil
		},
	}
}
