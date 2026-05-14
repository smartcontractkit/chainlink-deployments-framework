package mcms

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	chainsel "github.com/smartcontractkit/chain-selectors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	zapcoreobserver "go.uber.org/zap/zaptest/observer"

	cldf "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/changeset"
	cldfdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	cldfenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

func Test_newRunProposalHooksCmd(t *testing.T) { //nolint:paralleltest // test modifies shared state
	type testCase struct {
		name    string
		args    []string
		cfg     Config
		logs    *zapcoreobserver.ObservedLogs
		testDir string
		setup   func(*testing.T, *testCase)
		assert  func(*testing.T, *testCase, error)
	}
	noopSetup := func(*testing.T, *testCase) {}

	tests := []testCase{
		{
			name:  "failure: required flags not set",
			args:  []string{},
			cfg:   Config{},
			setup: noopSetup,
			assert: func(t *testing.T, _ *testCase, err error) {
				t.Helper()
				require.ErrorContains(t, err, `required flag(s) "environment", "proposal", "report", "selector" not set`)
			},
		},
		{
			name: "failure: no LoadChangesetFunction provided",
			args: []string{
				"--environment", "testnet",
				"--proposal", "proposal.json",
				"--report", "report.json",
				"--selector", strconv.FormatUint(chainsel.GETH_TESTNET.Selector, 10),
			},
			cfg:   Config{},
			setup: noopSetup,
			assert: func(t *testing.T, _ *testCase, err error) {
				t.Helper()
				require.ErrorContains(t, err, "load changesets function is required to run hooks")
			},
		},
		{
			name: "success: processes proposal without changesets without any side effects",
			cfg: Config{
				LoadChangesets: loadChangesets,
				Deps: Deps{
					EnvironmentLoader: func(
						ctx context.Context, domain cldfdomain.Domain, envKey string, lggr logger.Logger, opts ...cldfenv.LoadEnvironmentOption,
					) (cldf.Environment, error) {
						return cldf.Environment{
							Name:       envKey,
							Logger:     lggr,
							GetContext: func() context.Context { return ctx },
						}, nil
					},
				},
			},
			setup: func(t *testing.T, testCtx *testCase) {
				t.Helper()
				testCtx.testDir = t.TempDir()

				envName := "testnet"
				err := os.Mkdir(filepath.Join(testCtx.testDir, envName), 0o700)
				require.NoError(t, err)

				proposalPath := filepath.Join(testCtx.testDir, envName, "proposal.json")
				err = os.WriteFile(proposalPath, testProposalWithoutChangesetsJSON, 0o600)
				require.NoError(t, err)

				reportPath := filepath.Join(testCtx.testDir, envName, "report.json")
				err = os.WriteFile(reportPath, testReportJSON, 0o600)
				require.NoError(t, err)

				testCtx.args = []string{
					"--environment", envName,
					"--proposal", proposalPath,
					"--report", reportPath,
					"--selector", strconv.FormatUint(chainsel.GETH_TESTNET.Selector, 10),
				}

				lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)
				testCtx.cfg.Logger = lggr
				testCtx.logs = logs
			},
			assert: func(t *testing.T, testCtx *testCase, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, 0, testCtx.logs.FilterMessage("test-changeset-post-proposal-hook executed").Len())
				require.Equal(t, 0, testCtx.logs.FilterMessage("test-global-post-proposal-hook executed").Len())
			},
		},
		{
			name: "success: processes proposal with two changesets, only one of which has a hook",
			cfg: Config{
				LoadChangesets: loadChangesets,
				Deps: Deps{
					EnvironmentLoader: func(
						ctx context.Context, domain cldfdomain.Domain, envKey string, lggr logger.Logger, opts ...cldfenv.LoadEnvironmentOption,
					) (cldf.Environment, error) {
						return cldf.Environment{
							Name:       envKey,
							Logger:     lggr,
							GetContext: func() context.Context { return ctx },
						}, nil
					},
				},
			},
			setup: func(t *testing.T, testCtx *testCase) {
				t.Helper()
				generateStableUUIDs(t)

				testCtx.testDir = t.TempDir()

				envName := "testnet"
				err := os.Mkdir(filepath.Join(testCtx.testDir, envName), 0o700)
				require.NoError(t, err)

				proposalPath := filepath.Join(testCtx.testDir, envName, "proposal.json")
				err = os.WriteFile(proposalPath, testProposalWithChangesetsJSON, 0o600)
				require.NoError(t, err)

				reportPath := filepath.Join(testCtx.testDir, envName, "report.json")
				err = os.WriteFile(reportPath, testReportJSON, 0o600)
				require.NoError(t, err)

				testCtx.args = []string{
					"--environment", envName,
					"--proposal", proposalPath,
					"--report", reportPath,
					"--selector", strconv.FormatUint(chainsel.GETH_TESTNET.Selector, 10),
				}

				lggr, logs := logger.TestObserved(t, zapcore.DebugLevel)
				testCtx.cfg.Logger = lggr
				testCtx.logs = logs
			},
			assert: func(t *testing.T, testCtx *testCase, err error) {
				t.Helper()
				require.NoError(t, err)
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-changeset-post-proposal-hook executed").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-changeset-post-proposal-hook; # reports: 1").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-changeset-post-proposal-hook; config: map[config-key:config-value]").Len())
				require.Equal(t, 0, testCtx.logs.FilterMessage("test-changeset-post-proposal-hook; forkctx family: EVM").Len())
				require.Equal(t, 2, testCtx.logs.FilterMessage("test-global-post-proposal-hook executed").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-global-post-proposal-hook; # reports: 1").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-global-post-proposal-hook; # reports: 0").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-global-post-proposal-hook; config: map[config-key:config-value]").Len())
				require.Equal(t, 1, testCtx.logs.FilterMessage("test-global-post-proposal-hook; config: map[]").Len())
				require.Equal(t, 0, testCtx.logs.FilterMessage("test-global-post-proposal-hook; forkctx family: EVM").Len())
			},
		},
	}
	for _, tt := range tests { //nolint:paralleltest // test modifies shared state
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t, &tt)

			cmd := newRunProposalHooksCmd(tt.cfg)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			tt.assert(t, &tt, err)
		})
	}
}

// ----- helpers -----

var testProposalWithoutChangesetsJSON = []byte(`{
    "version": "v1",
    "kind": "TimelockProposal",
    "validUntil": 2004259681,
    "chainMetadata": {
        "3379446385462418246": {
            "mcmAddress": "0x0000000000000000000000000000000000000001",
            "startingOpCount": 0,
            "additionalFields": {}
        }
    },
    "description": "Test proposal",
    "overridePreviousRoot": false,
    "action": "schedule",
    "delay": "1h0m0s",
    "signatures": null,
    "timelockAddresses": {
        "3379446385462418246": "0x0000000000000000000000000000000000000002"
    },
    "operations": [
        {
            "chainSelector": 3379446385462418246,
            "transactions": [
                {
                    "to": "0x0000000000000000000000000000000000000000",
                    "additionalFields": {"value": 0},
                    "data": "ZGF0YQ==",
                    "contractType": "",
                    "tags": null
                }
            ]
        }
    ]
}`)

var testProposalWithChangesetsJSON = []byte(`{
	"version": "v1",
	"kind": "TimelockProposal",
	"validUntil": 2004259681,
	"chainMetadata": {
		"3379446385462418246": {
			"mcmAddress": "0x0000000000000000000000000000000000000001",
			"startingOpCount": 0,
			"additionalFields": {}
		}
	},
	"timelockAddresses": {
		"3379446385462418246": "0x0000000000000000000000000000000000000002"
	},
	"description": "Test proposal",
	"overridePreviousRoot": false,
	"action": "schedule",
	"delay": "1h0m0s",
	"signatures": null,
	"metadata": {
		"changesets": [
			{
				"id": "30377e5d-d431-4c27-ac79-31ced49fffc0",
				"name": "001_test_changeset",
				"input": {
					"input-key": "input-value"
				},
				"config": {
					"config-key": "config-value"
				}
			},
			{
				"id": "df1fbe75-5009-4561-b936-d3c1c6762c98",
				"name": "002_test_changeset",
				"input": {},
				"config": {}
			}
		]
	},
	"operations": [
		{
			"chainSelector": 3379446385462418246,
			"transactions": [
				{
					"to": "0x0000000000000000000000000000000000000003",
					"data": "ZGF0YQ==",
					"additionalFields": {
						"value": 0
					},
					"tags": [
						"test-tag",
						"changeset:30377e5d-d431-4c27-ac79-31ced49fffc0"
					]
				}
			]
		},
		{
			"chainSelector": 3379446385462418246,
			"transactions": [
				{
					"to": "0x0000000000000000000000000000000000000004",
					"data": "ZGF0YQ==",
					"additionalFields": {
						"value": 0
					},
					"tags": [
						"test-tag",
						"changeset:df1fbe75-5009-4561-b936-d3c1c6762c98"
					]
				}
			]
		}
	]
}`)

var testReportJSON = []byte(`[{
	"id": "f4a81ef3-54f2-46f0-82a8-b3954c355b5f",
	"status": "SUCCESS",
	"timestamp": "2026-03-25T23:18:23.394643-03:00",
	"input": {
		"index": 0,
		"operationID": "0x2a7a360a499fdbddead746aa11c1fbbf2b7ed1f2b86ee398fb4cad610e2ecb9d",
		"chainSelector": 3379446385462418246,
		"timelockAddress": "0xa316c2dEeaF7593F7E3Ce15D69D80eF60Aa1A919",
		"mcmAddress": "0xB09e94838Bd0c7c0ba105705Ec09ED6a10953EDe",
		"additionalFields": null,
		"changeset": {
			"id": "30377e5d-d431-4c27-ac79-31ced49fffc0",
			"index": 0,
			"name": "001_test_changeset"
		}
	},
	"output": {
		"transactionResult": {
			"hash": "0xda30e0c13e66e2fdec526620e5e655faa4d5de3139d0c04f7515d0cb3b145aab",
			"chainFamily": "evm",
			"rawData": {
				"type": "0x2",
				"chainId": "0x14a34",
				"nonce": "0x4a",
				"to": "0x6a08ed6cba5398f061eac2b3f01e0047974851d0",
				"gas": "0x1043c8",
				"gasPrice": null,
				"maxPriorityFeePerGas": "0xf4240",
				"maxFeePerGas": "0xa7d8c0",
				"value": "0x0",
				"input": "0x6ceef48000000000",
				"accessList": [],
				"v": "0x1",
				"r": "0x809f6092d7a9ac3c186be1b0faeba873f575561e8b7393bda57fc505f80b8932",
				"s": "0x51dded11cf4e94a99084d8c0d081835b73b7f3a00b049c467bcfd9b640ae047",
				"yParity": "0x1",
				"hash": "0xda30e0c13e66e2fdec526620e5e655faa4d5de3139d0c04f7515d0cb3b145aab"
			}
		}
	}
}]`)

func loadChangesets(envName string) (*changeset.ChangesetsRegistry, error) {
	registry := changeset.NewChangesetsRegistry()

	registry.Add("001_test_changeset",
		changeset.Configure(TestChangeset{}).
			With(testChangesetConfig{}).
			WithPostProposalHooks(changeset.PostProposalHook{
				HookDefinition: changeset.HookDefinition{
					Name:          "test-changeset-post-proposal-hook",
					Timeout:       30 * time.Second,
					FailurePolicy: changeset.Abort,
				},
				Func: func(ctx context.Context, params changeset.PostProposalHookParams) error {
					params.Env.Logger.Info("test-changeset-post-proposal-hook executed")
					params.Env.Logger.Infof("test-changeset-post-proposal-hook; # reports: %d", len(params.Reports))
					params.Env.Logger.Infof("test-changeset-post-proposal-hook; config: %v", params.Config)
					if params.Env.ForkContext != nil {
						params.Env.Logger.Infof("test-changeset-post-proposal-hook; forkctx family: %v",
							params.Env.ForkContext.ChainFamily())
					}

					return nil
				},
			}),
	)

	registry.Add("002_test_changeset",
		changeset.Configure(TestChangeset{}).
			With(testChangesetConfig{}))

	registry.AddGlobalPostProposalHooks(changeset.PostProposalHook{
		HookDefinition: changeset.HookDefinition{
			Name:          "test-global-post-proposal-hook",
			Timeout:       30 * time.Second,
			FailurePolicy: changeset.Abort,
		},
		Func: func(ctx context.Context, params changeset.PostProposalHookParams) error {
			params.Env.Logger.Info("test-global-post-proposal-hook executed")
			params.Env.Logger.Infof("test-global-post-proposal-hook; # reports: %d", len(params.Reports))
			params.Env.Logger.Infof("test-global-post-proposal-hook; config: %v", params.Config)
			if params.Env.ForkContext != nil {
				params.Env.Logger.Infof("test-global-post-proposal-hook; forkctx family: %v", params.Env.ForkContext.ChainFamily())
			}

			return nil
		},
	})

	return registry, nil
}

func generateStableUUIDs(t *testing.T) {
	t.Helper()
	uuid.SetRand(rand.New(rand.NewSource(1234)))
	t.Cleanup(func() { uuid.SetRand(nil) })
}

type testChangesetConfig struct{}

type TestChangeset struct{}

func (TestChangeset) Apply(env cldf.Environment, cfg testChangesetConfig) (cldf.ChangesetOutput, error) {
	return cldf.ChangesetOutput{}, nil
}

func (TestChangeset) VerifyPreconditions(env cldf.Environment, cfg testChangesetConfig) error {
	return nil
}
