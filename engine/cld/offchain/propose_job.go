package offchain

import (
	"context"
	"errors"
	"fmt"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"

	fdomain "github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	fpointer "github.com/smartcontractkit/chainlink-deployments-framework/internal/pointer"

	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
)

// ProposeJobRequest is the request to propose a job to a node using JD
type ProposeJobRequest struct {
	Job         string // toml
	Domain      fdomain.Domain
	Environment string
	// labels to filter nodes by
	NodeLabels map[string]string
	// labels to set on the new/updated job object
	JobLabels      map[string]string
	OffchainClient foffchain.Client
	Lggr           logger.Logger
}

// Validate validates the request
func (r ProposeJobRequest) Validate() error {
	if r.Job == "" {
		return errors.New("job is empty")
	}
	// TODO validate valid toml
	if r.Domain.Key() == "" {
		return errors.New("domain is empty")
	}
	if r.Environment == "" {
		return errors.New("environment is empty")
	}
	if r.OffchainClient == nil {
		return errors.New("offchain client is nil")
	}
	if r.Lggr == nil {
		return errors.New("logger is nil")
	}

	return nil
}

// ProposeJob proposes a job to a node using JD
func ProposeJob(ctx context.Context, req ProposeJobRequest) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	var merr error
	// always filter at least by product and environment
	domainKey := req.Domain.Key()
	selectors := []*ptypes.Selector{
		{
			Key:   "product",
			Op:    ptypes.SelectorOp_EQ,
			Value: &domainKey,
		},
		{
			Key:   "environment",
			Op:    ptypes.SelectorOp_EQ,
			Value: &req.Environment,
		},
	}
	for key, value := range req.NodeLabels {
		selectors = append(selectors, &ptypes.Selector{
			Key:   key,
			Op:    ptypes.SelectorOp_EQ,
			Value: fpointer.To(value), // TODO is this correct?
		})
	}
	nodes, err := req.OffchainClient.ListNodes(ctx, &nodev1.ListNodesRequest{Filter: &nodev1.ListNodesRequest_Filter{
		Enabled:   1,
		Selectors: selectors,
	}})
	if err != nil {
		return err
	}

	for _, node := range nodes.GetNodes() {
		_, err1 := req.OffchainClient.ProposeJob(ctx,
			&jobv1.ProposeJobRequest{
				NodeId: node.Id,
				Spec:   req.Job,
				Labels: convertLabels(req.JobLabels),
			})
		if err1 != nil {
			req.Lggr.Infow("Failed to propose job to node", "nodeId", node.Id, "nodeName", node.Name)
			merr = errors.Join(merr, fmt.Errorf("error proposing job to node %s spec %s : %w", node.Id, req.Job, err1))
		} else {
			req.Lggr.Infow("Successfully proposed job to node", "nodeId", node.Id, "nodeName", node.Name)
		}
	}

	return merr
}

// ProposeJobs proposes job specs to nodes using jobspecs file
// TODO remove when all migrations use Jobs instead of JobSpecs
func ProposeJobs(ctx context.Context, lggr logger.Logger, oc foffchain.Client, jobSpecFilePath string) error {
	jobSpecToNodes, err := fdomain.LoadJobSpecs(jobSpecFilePath) //nolint:staticcheck // TODO: remove when all migrations use Jobs instead of JobSpecs
	if err != nil {
		return err
	}

	for nodeID, jobs := range jobSpecToNodes {
		for _, job := range jobs {
			_, err1 := oc.ProposeJob(ctx,
				&jobv1.ProposeJobRequest{
					NodeId: nodeID,
					Spec:   job,
				})
			if err1 != nil {
				lggr.Infof("Failed to propose job to node %s\n", nodeID)

				err = errors.Join(err, fmt.Errorf(
					"error proposing job to node %s spec %s : %w", nodeID, job, err1),
				)
			} else {
				lggr.Infof("Successfully proposed job to node %s\n", nodeID)
			}
		}
	}

	return err
}

// ProposeWithJobDetails proposes job specs to nodes using jobspecs file
func ProposeWithJobDetails(ctx context.Context, lggr logger.Logger, oc foffchain.Client, jobsPath string) error {
	jobs, err := fdomain.LoadJobs(jobsPath)
	if err != nil {
		return err
	}

	for _, job := range jobs {
		_, err1 := oc.ProposeJob(ctx,
			&jobv1.ProposeJobRequest{
				NodeId: job.Node,
				Spec:   job.Spec,
			})
		if err1 != nil {
			lggr.Infof("Failed to propose job to node %s\n", job.Node)

			err = errors.Join(err, fmt.Errorf(
				"error proposing job to node %s spec %s : %w", job.Node, job, err1),
			)
		} else {
			lggr.Infof("Successfully proposed job to node %s\n", job.Node)
		}
	}

	return err
}

func convertLabels(labels map[string]string) []*ptypes.Label {
	res := make([]*ptypes.Label, 0, len(labels))
	for k, v := range labels {
		newVal := v
		res = append(res, &ptypes.Label{
			Key:   k,
			Value: &newVal,
		})
	}

	return res
}
