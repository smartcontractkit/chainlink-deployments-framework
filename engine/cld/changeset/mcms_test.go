package changeset

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/smartcontractkit/mcms"

	"github.com/stretchr/testify/require"
)

func Test_RunProposalHooks(t *testing.T) {
	t.Parallel()

	proposalHook := func(id string, execLog *[]string) PostProposalHook {
		return PostProposalHook{
			HookDefinition: HookDefinition{Name: "pp-hook", FailurePolicy: Abort},
			Func: func(_ context.Context, _ PostProposalHookParams) error {
				*execLog = append(*execLog, id)
				return nil
			},
		}
	}

	tests := []struct {
		name         string
		key          string
		setup        func(execLog *[]string) *ChangesetsRegistry
		wantExecLogs []string
		wantErr      string
	}{
		{
			name: "unknown key returns error",
			key:  "nonexistent",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				return NewChangesetsRegistry()
			},
			wantErr: "changeset 'nonexistent' not found",
		},
		{
			name: "per-changeset hook is invoked",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{
					changeset:         noopChangeset{},
					postProposalHooks: []PostProposalHook{proposalHook("proposal-changeset-hook", execLog)},
				}

				return r
			},
			wantExecLogs: []string{"proposal-changeset-hook"},
		},
		{
			name: "global hook is invoked",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{changeset: noopChangeset{}}
				r.AddGlobalPostProposalHooks(proposalHook("proposal-global-hook", execLog))

				return r
			},
			wantExecLogs: []string{"proposal-global-hook"},
		},
		{
			name: "changeset hook runs before global hook",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{
					changeset:         noopChangeset{},
					postProposalHooks: []PostProposalHook{proposalHook("proposal-changeset-hook", execLog)},
				}
				r.AddGlobalPostProposalHooks(proposalHook("proposal-global-hook", execLog))

				return r
			},
			wantExecLogs: []string{"proposal-changeset-hook", "proposal-global-hook"},
		},
		{
			name: "per-changeset hook Abort returns error",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{
					changeset: noopChangeset{},
					postProposalHooks: []PostProposalHook{{
						HookDefinition: HookDefinition{Name: "failing-hook", FailurePolicy: Abort},
						Func: func(_ context.Context, _ PostProposalHookParams) error {
							return errors.New("hook error")
						},
					}},
				}

				return r
			},
			wantErr: "post-proposal-hook \"failing-hook\" failed: hook error",
		},
		{
			name: "global hook Abort returns error",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{changeset: noopChangeset{}}
				r.AddGlobalPostProposalHooks(PostProposalHook{
					HookDefinition: HookDefinition{Name: "failing-global-hook", FailurePolicy: Abort},
					Func: func(_ context.Context, _ PostProposalHookParams) error {
						return errors.New("global hook error")
					},
				})

				return r
			},
			wantErr: "global post-proposal-hook \"failing-global-hook\" failed: global hook error",
		},
		{
			name: "Warn hook does not stop subsequent hooks",
			key:  "test-cs",
			setup: func(execLog *[]string) *ChangesetsRegistry {
				r := NewChangesetsRegistry()
				r.entries["test-cs"] = registryEntry{
					changeset: noopChangeset{},
					postProposalHooks: []PostProposalHook{
						{
							HookDefinition: HookDefinition{Name: "warn-hook", FailurePolicy: Warn},
							Func: func(_ context.Context, _ PostProposalHookParams) error {
								return errors.New("non-critical failure")
							},
						},
						proposalHook("second-hook", execLog),
					},
				}

				return r
			},
			wantExecLogs: []string{"second-hook"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			execLogs := []string{}
			registry := tt.setup(&execLogs)

			err := registry.RunProposalHooks(tt.key, hookTestEnv(t), &mcms.TimelockProposal{}, nil, nil)

			if tt.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tt.wantExecLogs, execLogs)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func Test_RunProposalHooks_HookReceivesCorrectParams(t *testing.T) {
	t.Parallel()

	proposal := &mcms.TimelockProposal{}
	input := "test-input"
	reports := []MCMSTimelockExecuteReport{{Type: MCMSTimelockExecuteReportType}}

	var receivedParams PostProposalHookParams

	r := NewChangesetsRegistry()
	r.entries["test-cs"] = registryEntry{
		changeset: noopChangeset{},
		postProposalHooks: []PostProposalHook{{
			HookDefinition: HookDefinition{Name: "param-checker", FailurePolicy: Warn},
			Func: func(_ context.Context, params PostProposalHookParams) error {
				receivedParams = params
				return nil
			},
		}},
	}

	err := r.RunProposalHooks("test-cs", hookTestEnv(t), proposal, input, reports)
	require.NoError(t, err)

	expectedParams := PostProposalHookParams{
		Env:          ProposalHookEnv{Name: "test-env"},
		ChangesetKey: "test-cs",
		Proposal:     proposal,
		Input:        input,
		Reports:      reports,
	}
	require.Empty(t, cmp.Diff(expectedParams, receivedParams,
		cmpopts.IgnoreFields(mcms.BaseProposal{}, "useSimulatedBackend"),
		cmpopts.IgnoreFields(ProposalHookEnv{}, "Logger", "BlockChains")))
}
