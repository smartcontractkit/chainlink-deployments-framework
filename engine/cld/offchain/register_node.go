package offchain

import (
	"context"
	"fmt"
	"sort"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"

	cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
)

// Matches the labels at
// https://github.com/smartcontractkit/chainlink/blob/b7f9f23f3aeae5d0cfd003c57bd1d9d19e2ddb80/deployment/environment/devenv/don.go#L38-L38
const (
	labelNodeTypeKey            = "type"
	labelNodeTypeValueBootstrap = "bootstrap"
	labelNodeTypeValuePlugin    = "plugin"
)

// RegisterNode registers a single node with the job distributor. It errors if the node is already registered.
func RegisterNode(
	ctx context.Context,
	jd cldf_offchain.Client,
	name string,
	csaKey string,
	isBootstrap bool,
	domain domain.Domain,
	environment string,
	extraLabels map[string]string,
) (string, error) {
	domainKey := domain.Key()
	labels := make([]*ptypes.Label, 0)
	labels = append(labels, &ptypes.Label{
		Key:   "product",
		Value: &domainKey,
	})
	labels = append(labels, &ptypes.Label{
		Key:   "environment",
		Value: &environment,
	})

	// Sort extraLabels keys to ensure deterministic label ordering
	extraLabelKeys := make([]string, 0, len(extraLabels))
	for key := range extraLabels {
		extraLabelKeys = append(extraLabelKeys, key)
	}
	sort.Strings(extraLabelKeys)

	for _, key := range extraLabelKeys {
		labels = append(labels, &ptypes.Label{
			Key:   key,
			Value: pointer.To(extraLabels[key]),
		})
	}
	if isBootstrap {
		labels = append(labels, &ptypes.Label{
			Key:   labelNodeTypeKey,
			Value: pointer.To(labelNodeTypeValueBootstrap),
		})
	} else {
		labels = append(labels, &ptypes.Label{
			Key:   labelNodeTypeKey,
			Value: pointer.To(labelNodeTypeValuePlugin),
		})
	}
	resp, err := jd.RegisterNode(ctx, &nodev1.RegisterNodeRequest{
		Name:      name,
		PublicKey: csaKey,
		Labels:    labels,
	})
	if err != nil {
		return "", fmt.Errorf("failed to register node %s : %w", name, err)
	}
	if resp == nil || resp.Node == nil || resp.Node.Id == "" {
		return "", fmt.Errorf("failed to register node %s, blank response received", name)
	}

	return resp.Node.Id, nil
}
