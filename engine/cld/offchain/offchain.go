package offchain

import (
	"context"
	"fmt"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	csav1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/csa"
	"golang.org/x/oauth2"

	cfgenv "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config/env"
	enginedomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	credentials "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	jdoffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd"
	jdprov "github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/provider"
)

// LoadOffchainClient loads an offchain client for the specified domain and environment.
func LoadOffchainClient(
	ctx context.Context,
	domain enginedomain.Domain,
	env string,
	config *cfgenv.Config,
	lggr logger.Logger,
	useRealBackends bool,
) (foffchain.Client, error) {
	var jd foffchain.Client

	endpoints := config.Offchain.JobDistributor.Endpoints
	auth := config.Offchain.JobDistributor.Auth

	if domain.Key() == enginedomain.MustGetDomain("keystone").Key() && endpoints.WSRPC == "" {
		lggr.Warn("Skipping JD initialization for Keystone, fallback to CLO data")
	} else if endpoints.WSRPC == "" || endpoints.GRPC == "" {
		lggr.Warn("Skipping JD initialization no JD config for WS or gRPC")
	} else if endpoints.WSRPC != "" && endpoints.GRPC != "" {
		lggr.Info("Initializing JD client")
		var oauth oauth2.TokenSource
		if config.Offchain.JobDistributor.Auth != nil {
			source := jdoffchain.NewCognitoTokenSource(jdoffchain.CognitoAuth{
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
		creds := credentials.GetCredsForEnv(env)

		var offchainOptions []jdprov.ClientProviderOption
		if !useRealBackends {
			lggr.Infow("Using a dry-run JD client")
			offchainOptions = append(offchainOptions, jdprov.WithDryRun(lggr))
		}

		provider := jdprov.NewClientOffchainProvider(jdprov.ClientOffchainProviderConfig{
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
