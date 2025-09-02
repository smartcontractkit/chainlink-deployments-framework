package offchain

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	"golang.org/x/oauth2"

	cldf_config_env "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	cldf_domain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal"
	cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	offchain_jd "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
	offchain_jd_provider "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
)

// LoadOffchainClient loads an offchain client for the specified domain and environment.
func LoadOffchainClient(
	ctx context.Context,
	domain cldf_domain.Domain,
	env string,
	config *cldf_config_env.Config,
	lggr logger.Logger,
	useRealBackends bool,
) (cldf_offchain.Client, error) {
	var jd cldf_offchain.Client

	endpoints := config.Offchain.JobDistributor.Endpoints
	auth := config.Offchain.JobDistributor.Auth

	if domain.Key() == cldf_domain.MustGetDomain("keystone").Key() && endpoints.WSRPC == "" {
		lggr.Warn("Skipping JD initialization for Keystone, fallback to CLO data")
	} else if endpoints.WSRPC == "" || endpoints.GRPC == "" {
		lggr.Warn("Skipping JD initialization no JD config for WS or gRPC")
	} else if endpoints.WSRPC != "" && endpoints.GRPC != "" {
		lggr.Info("Initializing JD client")
		var oauth oauth2.TokenSource
		if config.Offchain.JobDistributor.Auth != nil {
			source := offchain_jd.NewCognitoTokenSource(offchain_jd.CognitoAuth{
				AppClientID:     auth.CognitoAppClientID,
				AppClientSecret: auth.CognitoAppClientSecret,
				Username:        auth.Username,
				Password:        auth.Password,
				AWSRegion:       auth.AWSRegion,
			})
			if err := source.Authenticate(ctx); err != nil {
				return nil, err
			}

			oauth = source
		}
		creds := internal.GetCredsForEnv(env)

		var offchainOptions []offchain_jd_provider.ClientProviderOption
		if !useRealBackends {
			lggr.Infow("Using a dry-run JD client")
			offchainOptions = append(offchainOptions, offchain_jd_provider.WithDryRun(lggr))
		}

		provider := offchain_jd_provider.NewClientOffchainProvider(offchain_jd_provider.ClientOffchainProviderConfig{
			GRPC:  endpoints.GRPC,
			WSRPC: endpoints.WSRPC,
			Creds: creds,
			Auth:  oauth,
		}, offchainOptions...)

		var err error
		jd, err = provider.Initialize(ctx)
		if err != nil {
			return nil, err
		}

		var kp *csav1.ListKeypairsResponse
		kp, err = jd.ListKeypairs(ctx, &csav1.ListKeypairsRequest{})
		if err != nil {
			return jd, fmt.Errorf("unable to reach the JD instance %s. Are you on the VPN? %w", endpoints.GRPC, err)
		}
		lggr.Debugw("JD CSA Key", "key", kp.Keypairs[0].PublicKey)
	}

	return jd, nil
}
