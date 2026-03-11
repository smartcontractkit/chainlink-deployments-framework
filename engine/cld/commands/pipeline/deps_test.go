package pipeline

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/environment"
)

func TestDeps_applyDefaults(t *testing.T) {
	t.Parallel()

	d := &Deps{}
	require.Nil(t, d.EnvironmentLoader)

	d.applyDefaults()
	require.NotNil(t, d.EnvironmentLoader)
}

func TestDeps_applyDefaults_PreservesCustomLoader(t *testing.T) {
	t.Parallel()

	customLoader := func(context.Context, domain.Domain, string, ...environment.LoadEnvironmentOption) (fdeployment.Environment, error) {
		return fdeployment.Environment{}, nil
	}
	d := &Deps{EnvironmentLoader: customLoader}

	d.applyDefaults()
	require.NotNil(t, d.EnvironmentLoader)
	// Should preserve the custom loader, not override
	env, err := d.EnvironmentLoader(t.Context(), domain.NewDomain(t.TempDir(), "test"), "testnet")
	require.NoError(t, err)
	require.NotNil(t, env)
}
