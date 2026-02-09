package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"
	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
	"github.com/spf13/cobra"

	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"

	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/config"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/domain"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/internal/credentials"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/legacy/cli"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/offchain"
	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	fnode "github.com/smartcontractkit/chainlink-deployments-framework/offchain/node"
)

// defaultTimeout governs operations that may block
var defaultTimeout = time.Second * 180

// NewJDCmds holds the commands related to job distributor.
func (c Commands) NewJDCmds(domain domain.Domain) *cobra.Command {
	jdCmd := &cobra.Command{
		Use:   "jd",
		Short: "Manage job distributor interactions",
	}

	nodeCmd := &cobra.Command{
		Use:   "node",
		Short: "Manage nodes",
	}
	nodeCmd.AddCommand(
		c.newJDNodeList(domain),
		c.newJDNodeInspect(domain),
		c.newJDNodePatchLabels(domain),
		c.newJDNodeRegister(domain),
		c.newJDNodeBatchRegister(domain),
		c.newJDNodeUpdate(domain),
		c.newJDNodeBatchUpdate(domain),
		c.newJDNodeSaveAll(domain))

	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Manage jobs",
	}
	jobCmd.AddCommand(
		c.newJDJobPropose(domain),
		c.newJDJobBatchPropose(domain),
	)

	jdCmd.AddCommand(nodeCmd, jobCmd)

	jdCmd.PersistentFlags().
		StringP("environment", "e", "", "Environment (required)")
	_ = jdCmd.MarkPersistentFlagRequired("environment")

	return jdCmd
}

var (
	// node list cmd
	jdNodeListLong = cli.LongDesc(`
	List nodes registered in the job distributor with optional filters.
	Supports label filters, DON filtering, JSON or table output.
	`)
	jdNodeListExample = cli.Examples(`
	# List all nodes in JSON
	cld jd node list -e prod -f json

	# Filter by label
	cld jd node list -e prod -l region=us-east
	`)
)

// newJDNodeList lists nodes registered with the job distributor with optional filters.
func (c Commands) newJDNodeList(domain domain.Domain) *cobra.Command {
	var (
		labels       []string
		dons         []string
		format       string
		viewJobs     bool
		validFormats = []string{"table", "json"}
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Long:    jdNodeListLong,
		Example: jdNodeListExample,
		Short:   "List out nodes registered with job distributor",
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			c.lggr.Infof("Listing nodes for job-distributor for %s in environment: %s", domain, envKey)
			format = strings.ToLower(format)
			if !slices.Contains(validFormats, format) {
				return fmt.Errorf("invalid format '%s'", format)
			}

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			filterLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}

			domainKey := domain.Key()
			selectors := []*ptypes.Selector{
				{
					Key:   "product",
					Op:    ptypes.SelectorOp_EQ,
					Value: &domainKey,
				},
				{
					Key:   "environment",
					Op:    ptypes.SelectorOp_EQ,
					Value: &envKey,
				},
			}
			for key, value := range filterLabels {
				selectors = append(selectors, &ptypes.Selector{
					Key:   key,
					Op:    ptypes.SelectorOp_EQ, // TODO: support IN
					Value: &value,
				})
			}

			names := make([]string, 0)
			for _, name := range dons {
				// adding prefix to align with registration changeset
				donName := "don-" + name
				selectors = append(selectors, &ptypes.Selector{
					Key: donName,
					Op:  ptypes.SelectorOp_EXIST,
				})
				names = append(names, donName)
			}
			if len(names) > 0 {
				c.lggr.Infof("filtering for DONs %v", names)
			}

			nodes, err := offChainClient.ListNodes(cmd.Context(), &nodev1.ListNodesRequest{
				Filter: &nodev1.ListNodesRequest_Filter{
					Selectors: selectors,
				},
			})
			if err != nil {
				return err
			}

			var (
				nodeIdToJobs      = make(map[string][]*jobv1.Job)
				nodeIdToProposals = make(map[string][]*jobv1.Proposal)
			)

			if viewJobs {
				nodeIdToJobs, err = listJobsByNodeId(cmd.Context(), offChainClient, toNodeIDs(nodes.GetNodes()))
				if err != nil {
					return err
				}

				nodeIdToProposals, err = listProposalsByNodeId(cmd.Context(), offChainClient, nodeIdToJobs)
				if err != nil {
					return err
				}
			}

			nv := toNodeViews(nodes.GetNodes(), nodeIdToProposals, nodeIdToJobs)

			switch format {
			case "table":
				// TODO: update writeNodeTable to support jobs view
				writeNodeTable(nodes.GetNodes())
			case "json":
				b, err := json.MarshalIndent(nv, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal nodes: %w", err)
				}
				cmd.Println(string(b))
			default:
				return fmt.Errorf("invalid format '%s'", format)
			}

			return nil
		}}

	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")
	cmd.Flags().StringSliceVar(&dons, "dons", nil, "DON Names to filter by")
	cmd.Flags().BoolVar(&viewJobs, "view-jobs", false, "Enable to view the status of job proposals for selected nodes.")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format ["+strings.Join(validFormats, "|")+"]")

	return cmd
}

// NodeView is a view of a node that optionally includes a map of jobs.
type NodeView struct {
	*nodev1.Node
	// CreatedAt is the creation timestamp of the node
	CreatedAt string `json:"created_at"`
	// UpdatedAt is the last updated timestamp of the node
	UpdatedAt string `json:"updated_at"`

	// Jobs maps a job UUID to a list of proposal views
	Jobs map[string][]ProposalView `json:"jobs,omitempty"`
}

// ProposalView is the viewable version of a proposal.
type ProposalView struct {
	ID                 string `json:"id"`
	JobID              string `json:"job_id"`
	Status             string `json:"status"`
	Revision           int64  `json:"revision"`
	DeliveryStatus     string `json:"delivery_status"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	AckedAt            string `json:"acked_at"`
	ResponseReceivedAt string `json:"response_received_at"`
}

var (
	// node inspect cmd
	jdNodeInspectLong = cli.LongDesc(`
	Inspect chain configs for specific node IDs in JD.
	Outputs in table or JSON format.
	`)
	jdNodeInspectExample = cli.Examples(`
	# Inspect chain configs for node1
	cld jd node inspect -e staging node1
	`)
)

func (c Commands) newJDNodeInspect(domain domain.Domain) *cobra.Command {
	var (
		labels       []string
		format       string
		validFormats = []string{"table", "json"}
	)

	cmd := &cobra.Command{
		Use:     "inspect",
		Aliases: []string{"i"},
		Short:   "Inspect chain configs for chainlink node(s) with job-distributor",
		Long:    jdNodeInspectLong,
		Example: jdNodeInspectExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			c.lggr.Infof("Inspecting chain configs for job-distributor for %s in environment: %s\n", domain, envKey)
			if !slices.Contains(validFormats, format) {
				return fmt.Errorf("invalid format '%s'", format)
			}

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			filterLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}

			domainKey := domain.Key()
			selectors := []*ptypes.Selector{
				{
					Key:   "product",
					Op:    ptypes.SelectorOp_EQ,
					Value: &domainKey,
				},
				{
					Key:   "environment",
					Op:    ptypes.SelectorOp_EQ,
					Value: &envKey,
				},
			}
			for key, value := range filterLabels {
				selectors = append(selectors, &ptypes.Selector{
					Key:   key,
					Op:    ptypes.SelectorOp_EQ, // TODO: support IN
					Value: &value,
				})
			}

			// TODO: if using label, filter nodes by labels first, then use those node ids
			// nodes, err := env.Offchain.ListNodes(cmd.Context(), &nodev1.ListNodesRequest{
			// 	Filter: &nodev1.ListNodesRequest_Filter{
			// 		Selectors: selectors,
			// 	},
			// })
			// if err != nil {
			// 	return err
			// }

			chainConfigs, err := offChainClient.ListNodeChainConfigs(cmd.Context(), &nodev1.ListNodeChainConfigsRequest{
				Filter: &nodev1.ListNodeChainConfigsRequest_Filter{
					NodeIds: args,
				},
			})
			if err != nil {
				return err
			}
			configsByNode := make(map[string][]*nodev1.ChainConfig)

			for _, config := range chainConfigs.GetChainConfigs() {
				configsByNode[config.NodeId] = append(configsByNode[config.NodeId], config)
			}

			switch format {
			case "table":
				writeChainConfigTable(configsByNode)
			case "json":
				b, err := json.MarshalIndent(configsByNode, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal chainConfigs: %w", err)
				}
				cmd.Println(string(b))
			default:
				return fmt.Errorf("invalid format '%s'", format)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")
	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format ["+strings.Join(validFormats, "|")+"]")

	return cmd
}

var (
	// patch labels cmd
	jdNodePatchLong = cli.LongDesc(`
	Patch or update labels on existing JD nodes.
	Existing labels remain unless overwritten or cleared.
	`)
	jdNodePatchExample = cli.Examples(`
	# Patch label on node1
	cld jd node labels-patch -e prod --label region=eu node1
	`)
)

func (c Commands) newJDNodePatchLabels(domain domain.Domain) *cobra.Command {
	var (
		labels []string
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:     "labels-patch",
		Aliases: []string{},
		Short:   "Patch labels for nodes in job-distributor",
		Long:    jdNodePatchLong,
		Example: jdNodePatchExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			c.lggr.Infof("Patch labels for nodes for %s in environment: %s\n", domain, envKey)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			newLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}

			domainKey := domain.Key()
			selectors := []*ptypes.Selector{
				{
					Key:   "product",
					Op:    ptypes.SelectorOp_EQ,
					Value: &domainKey,
				},
				{
					Key:   "environment",
					Op:    ptypes.SelectorOp_EQ,
					Value: &envKey,
				},
			}

			nodeIDs := args
			if len(nodeIDs) == 0 {
				return errors.New("no node ids specified")
			}
			// if nodeIDs starts with `p2p_` lookup by p2p_id instead
			filterByPeerIDs := strings.HasPrefix(nodeIDs[0], "p2p_")
			var filter *nodev1.ListNodesRequest_Filter
			if filterByPeerIDs {
				selector := strings.Join(nodeIDs, ",")
				filter = &nodev1.ListNodesRequest_Filter{
					Selectors: append(selectors, &ptypes.Selector{
						Key:   "p2p_id",
						Op:    ptypes.SelectorOp_IN,
						Value: &selector,
					}),
				}
			} else {
				filter = &nodev1.ListNodesRequest_Filter{
					Ids:       nodeIDs,
					Selectors: selectors,
				}
			}

			nodes, err := offChainClient.ListNodes(cmd.Context(), &nodev1.ListNodesRequest{
				Filter: filter,
			})
			if err != nil {
				return err
			}
			if len(nodeIDs) != len(nodes.Nodes) {
				return errors.New("some node ids not found")
			}

			for _, node := range nodes.GetNodes() {
				labels := node.Labels
				for key, value := range newLabels {
					index := slices.IndexFunc(node.Labels, func(l *ptypes.Label) bool { return l.Key == key })
					if value == "" {
						if index >= 0 {
							// remove existing value
							labels = append(labels[:index], labels[index+1:]...)
						}
					} else {
						if index >= 0 {
							// update existing value
							labels[index].Value = &value
						} else {
							// add new value
							labels = append(labels, &ptypes.Label{Key: key, Value: &value})
						}
					}
				}

				c.lggr.Infof("Updating labels for node %s: %+v\n", node.Id, labels)

				if dryRun {
					continue
				}
				_, err := offChainClient.UpdateNode(cmd.Context(), &nodev1.UpdateNodeRequest{
					Id:        node.Id,
					Name:      node.Name,
					PublicKey: node.PublicKey,
					Labels:    labels,
				})
				if err != nil {
					return fmt.Errorf("failed to update labels: %w", err)
				}
			}

			return nil
		}}

	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run, only log updated state")

	return cmd
}

var (
	jdNodeRegisterLong = cli.LongDesc(`
	Register a new Chainlink node in the job distributor.
	Requires node name and CSA address.
	`)
	jdNodeRegisterExample = cli.Examples(`
	# Register node with bootstrap
	cld jd node register -e dev -n mynode -a 0xabc... --bootstrap
	`)
)

func (c Commands) newJDNodeRegister(domain domain.Domain) *cobra.Command {
	var (
		name        string
		csaAddress  string
		isBootstrap bool
		labels      []string
	)

	cmd := &cobra.Command{
		Use:     "register",
		Short:   "Register chainlink node with job-distributor",
		Long:    jdNodeRegisterLong,
		Example: jdNodeRegisterExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			envdir := domain.EnvDir(envKey)

			c.lggr.Infof("Register node to job-distributor for %s in environment: %s\n",
				domain, envKey,
			)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			extraLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}

			// TODO: figure out what happens if we register the same node multiple times
			id, err := offchain.RegisterNode(
				cmd.Context(), offChainClient, name, csaAddress, isBootstrap, domain, envKey, extraLabels,
			)
			if err != nil {
				return err
			}

			c.lggr.Infof("Node %s registered with id: %s\n", name, id)

			if err = envdir.SaveNodes([]string{id}); err != nil {
				return err
			}

			c.lggr.Infof("Node ID saved to nops_id for %s in environment: %s\n",
				domain, envKey,
			)

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Node name (required)")
	cmd.Flags().StringVarP(&csaAddress, "csa-address", "a", "", "CSA address (required)")
	cmd.Flags().BoolVarP(&isBootstrap, "bootstrap", "b", false, "IsBootstrap")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("csa-address")

	return cmd
}

var (
	jdNodeBatchRegisterLong = cli.LongDesc(`
	Batch register multiple nodes from a JSON config file.
	The config must contain a serialized NodeSet.
	`)
	jdNodeBatchRegisterExample = cli.Examples(`
	# Batch register nodes
	cld jd node batch-register -e prod -d nodes.json
	`)
)

// newJDNodeBatchRegister registers multiple nodes with the job distributor
func (c Commands) newJDNodeBatchRegister(domain domain.Domain) *cobra.Command {
	var (
		pth    string
		labels []string
	)

	cmd := &cobra.Command{
		Use:     "batch-register",
		Short:   "Register chainlink nodes with job-distributor",
		Long:    jdNodeBatchRegisterLong,
		Example: jdNodeBatchRegisterExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			if domain.Key() == "" {
				return errors.New("domain is empty")
			}

			envdir := domain.EnvDir(envKey)

			c.lggr.Infof("Register nodes to job-distributor for %s in environment: %s\n",
				domain, envKey,
			)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), defaultTimeout)
			defer cancel()

			b, err := os.ReadFile(pth)
			if err != nil {
				return fmt.Errorf("failed to read file %s : %w", pth, err)
			}
			var nodeSet NodeSet
			err = json.Unmarshal(b, &nodeSet)
			if err != nil {
				return fmt.Errorf("failed to unmarshal file %s : %w", pth, err)
			}
			extraLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}
			var ids []string
			// loop over nodes and register them with the
			for _, n := range nodeSet.Nodes {
				labels := n.Labels()
				for k, v := range extraLabels {
					labels[k] = v
				}

				// TODO: figure out what happens if we register the same node multiple times
				id, err1 := offchain.RegisterNode(ctx, offChainClient, n.Name, n.CSAKey, n.IsBootstrap(), domain, envKey, labels)
				if err1 != nil {
					return err1
				}
				c.lggr.Infof("Node %s registered with id: %s\n", n.Name, id)
				ids = append(ids, id)
			}

			err1 := envdir.SaveNodes(ids)
			if err1 != nil {
				return err1
			}

			c.lggr.Infof("Node IDs saved to nops_id for %s in environment: %s\n", domain, envKey)

			return nil
		},
	}

	cmd.Flags().StringVarP(&pth, "config", "d", "", "Path to the config file. Config file must contain serialized `NodeSet`")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")

	_ = cmd.MarkFlagRequired("config")

	return cmd
}

var (
	jdJobProposeLong = cli.LongDesc(`
	Propose a single job spec to matching JD nodes.
	Filters by labels if provided.
	`)
	jdJobProposeExample = cli.Examples(`
	# Propose job spec
	cld jd job propose -e prod -j ./job.json
	`)
)

func (c Commands) newJDJobPropose(domain domain.Domain) *cobra.Command {
	var (
		jobspecPath string
		labels      []string
	)

	cmd := &cobra.Command{
		Use:     "propose",
		Short:   "Propose a single job to multiple nodes",
		Long:    jdJobProposeLong,
		Example: jdJobProposeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			c.lggr.Infof("Propose job to nodes for %s in environment: %s", domain, envKey)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			filterLabels, err := parseLabelsArg(labels)
			if err != nil {
				return fmt.Errorf("failed to parse labels: %w", err)
			}
			job, err := os.ReadFile(jobspecPath)
			if err != nil {
				return fmt.Errorf("failed to read job spec file %s : %w", jobspecPath, err)
			}

			r := offchain.ProposeJobRequest{
				Job:            string(job),
				Domain:         domain,
				Environment:    envKey,
				NodeLabels:     filterLabels,
				OffchainClient: offChainClient,
				Lggr:           c.lggr,
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 180*time.Second)
			defer cancel()

			return offchain.ProposeJob(ctx, r)
		},
	}

	cmd.Flags().StringVarP(&jobspecPath, "jobspec", "j", "", "Absolute file path containing the jobspec to be proposed")
	cmd.Flags().StringSliceVarP(&labels, "label", "l", nil, "Labels (key=value)")

	_ = cmd.MarkFlagRequired("jobspec")

	return cmd
}

var (
	jdNodeUpdateLong = cli.LongDesc(`
	Update a registered JD node’s properties by node ID.
	Supports updating name, public key, or labels.
	`)
	jdNodeUpdateExample = cli.Examples(`
	# Update CSA key for a node
	cld jd node update -e staging -i node1 -k csa-key -v 0xdef...
	`)
)

func (c Commands) newJDNodeUpdate(domain domain.Domain) *cobra.Command {
	var (
		nodeID        string
		keysToUpdate  []string
		keyValues     []string
		labelKeys     []string
		labelValues   []string
		validKeyTypes = []string{string(offchain.NodeKey_CSAKey), string(offchain.NodeKey_Name), string(offchain.NodeKey_Label)}
	)
	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update single chainlink node with job-distributor when the node id is known",
		Long:    jdNodeUpdateLong,
		Example: jdNodeUpdateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			if domain.Key() == "" {
				return errors.New("domain is empty")
			}

			c.lggr.Infof("Updating node %s in job-distributor for %s in environment %s with key-type: %v, key-value: %v labels: %v label-values:%s \n ",
				nodeID, domain, envKey, keysToUpdate, keyValues, labelKeys, labelValues)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), defaultTimeout)
			defer cancel()
			resp, err := offChainClient.GetNode(ctx, &nodev1.GetNodeRequest{
				Id: nodeID,
			})
			if err != nil || resp == nil || resp.Node == nil {
				return fmt.Errorf("failed to list nodes with id %s: %w", nodeID, err)
			}

			r := &nodev1.UpdateNodeRequest{
				Id:        nodeID,
				Name:      resp.Node.Name,
				PublicKey: resp.Node.PublicKey,
				Labels:    resp.Node.Labels,
			}
			for i, keyToUpdate := range keysToUpdate {
				if !slices.Contains(validKeyTypes, keyToUpdate) {
					return fmt.Errorf("invalid key-type '%s'", keyToUpdate)
				}
				switch keyToUpdate {
				case string(offchain.NodeKey_CSAKey):
					r.PublicKey = keyValues[i]
				case string(offchain.NodeKey_Name):
					r.Name = keyValues[i]
				case string(offchain.NodeKey_Label):
					if len(labelKeys) == 0 {
						r.Labels = []*ptypes.Label{}
					}
					if len(labelKeys) != len(labelValues) {
						return errors.New("label-keys and label-values must be of the same length")
					}
					for i, k := range labelKeys {
						r.Labels = append(r.Labels, &ptypes.Label{Key: k, Value: &labelValues[i]})
					}
				}
			}

			_, err = offChainClient.UpdateNode(ctx, r)
			if err != nil {
				return fmt.Errorf("failed to update node %s with request %+v: %w", nodeID, r, err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&nodeID, "nodeID", "i", "", "Node ID as registered in Job Distributor (required)")
	cmd.Flags().StringSliceVarP(&keysToUpdate, "key-types", "k", []string{}, "Key type to update in the node ["+strings.Join(validKeyTypes, "|")+"]")
	cmd.Flags().StringSliceVarP(&keyValues, "key-values", "v", []string{}, "Value to update the key with, should be in the same order as key-types,"+
		" use label-keys and label-values for label")
	cmd.Flags().StringSliceVarP(&labelKeys, "label-keys", "l", []string{}, "Label keys to update, mandatory if key-type is label")
	cmd.Flags().StringSliceVarP(&labelValues, "label-values", "m", []string{}, "Label values to update, mandatory if key-type is label")

	_ = cmd.MarkFlagRequired("nodeID")
	_ = cmd.MarkFlagRequired("key-types")

	return cmd
}

var (
	// batch update nodes cmd
	jdNodeBatchUpdateLong = cli.LongDesc(`
	Batch-update JD nodes via a TOML config file.
	Supports key-type based updates across multiple nodes.
	`)
	jdNodeBatchUpdateExample = cli.Examples(`
	# Batch update label-key=region for nodes
	cld jd node batch-update -e prod -d config.toml -k label -l region
	`)
)

// newJDNodeBatchUpdate updates multiple nodes with the job distributor
func (c Commands) newJDNodeBatchUpdate(domain domain.Domain) *cobra.Command {
	var (
		pth           string
		keyType       string
		labelKey      string
		validKeyTypes = []string{string(offchain.NodeKey_CSAKey), string(offchain.NodeKey_Name), string(offchain.NodeKey_Label)}
	)

	cmd := &cobra.Command{
		Use:     "batch-update",
		Short:   "Update chainlink nodes with job-distributor",
		Long:    jdNodeBatchUpdateLong,
		Example: jdNodeBatchUpdateExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			if domain.Key() == "" {
				return errors.New("domain is empty")
			}

			if !slices.Contains(validKeyTypes, keyType) {
				return fmt.Errorf("invalid key-type '%s'", keyType)
			}

			c.lggr.Infof("Updating nodes in job-distributor for %s in environment %s using key-type: %s \n", domain, envKey, keyType)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), defaultTimeout)
			defer cancel()

			b, err := os.ReadFile(pth)
			if err != nil {
				return fmt.Errorf("failed to read file %s : %w", pth, err)
			}
			var nodeSet NodeSet
			err = toml.Unmarshal(b, &nodeSet)
			if err != nil {
				return fmt.Errorf("failed to unmarshal file %s : %w", pth, err)
			}

			// loop over nodes and register them with the
			var updateRequests []*offchain.UpdateNodeRequest
			for _, n := range nodeSet.Nodes {
				var r *offchain.UpdateNodeRequest
				r, err = offchain.NewUpdateNodeRequest(n, offchain.NodeFinderCfg{
					KeyType:   offchain.NodeKey(keyType),
					LabelName: &labelKey,
				})
				if err != nil {
					return fmt.Errorf("failed to create node key config: %w", err)
				}
				updateRequests = append(updateRequests, r)
			}
			err = offchain.UpdateNodes(ctx, offChainClient, offchain.UpdateNodesRequest{Requests: updateRequests})
			if err != nil {
				return fmt.Errorf("failed to update nodes: %w", err)
			}
			for _, r := range updateRequests {
				c.lggr.Infof("Updated node: %s\n", r.NodeKeyCriteria())
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&pth, "config", "d", "", "Path to the config file. Config file is TOML for Nodes")
	cmd.Flags().StringVarP(&keyType, "key-type", "k", string(offchain.NodeKey_CSAKey), "Key type to use to find a node ["+strings.Join(validKeyTypes, "|")+"]")
	cmd.Flags().StringVarP(&labelKey, "label-key", "l", "", "Label key to use to find a node when key-type is label. Required if key-type is label")

	_ = cmd.MarkFlagRequired("config")

	return cmd
}

var (
	jdJobBatchProposeLong = cli.LongDesc(`
	Propose multiple jobs in bulk either from a jobspec file or a jobs directory.
	Mutually exclusive flags --jobspec or --jobs.
	`)
	jdJobBatchProposeExample = cli.Examples(`
	# Batch propose from jobspec
	cld jd job batch-propose -e prod -j specs.json
	`)
)

func (c Commands) newJDJobBatchPropose(domain domain.Domain) *cobra.Command {
	var (
		jobspecPath string
		jobsPath    string
	)

	cmd := &cobra.Command{
		Use:     "batch-propose",
		Short:   "Propose all jobs in a jobspecs artifact to nodes",
		Long:    jdJobBatchProposeLong,
		Example: jdJobBatchProposeExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			jobsPathExists := jobsPath != ""
			jobspecPathExists := jobspecPath != ""
			if jobsPathExists == jobspecPathExists {
				return errors.New("must provide exactly one of jobspec or jobs")
			}

			c.lggr.Infof("Propose job to offchain components for %s in environment: %s\n", domain, envKey)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			if jobspecPathExists {
				return offchain.ProposeJobs(cmd.Context(), c.lggr, offChainClient, jobspecPath)
			}

			return offchain.ProposeWithJobDetails(cmd.Context(), c.lggr, offChainClient, jobsPath)
		},
	}

	cmd.Flags().StringVarP(&jobspecPath, "jobspec", "j", "", "Absolute file path containing the jobspec to be proposed")
	cmd.Flags().StringVarP(&jobsPath, "jobs", "p", "", "Absolute file path containing the job details to be proposed")
	// TODO mark jobs as required when all changesets use jobs instead of jobspecs

	return cmd
}

var (
	jdNodeSaveAllLong = cli.LongDesc(`
	Fetch all JD nodes and update the local nodes.json file.
	Use --dry-run to preview changes.
	`)
	jdNodeSaveAllExample = cli.Examples(`
	# Save nodes.json
	cld jd node save-all -e prod
	`)
)

// newJDNodeSaveAll builds a command that fetches node data from job distributor
// and updates the nodes.json file with the current nodes
func (c Commands) newJDNodeSaveAll(domain domain.Domain) *cobra.Command {
	var (
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:     "save-all",
		Short:   "Recreate the nodes.json",
		Long:    jdNodeSaveAllLong,
		Example: jdNodeSaveAllExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			envKey, _ := cmd.Flags().GetString("environment")
			c.lggr.Infof("Updating nodes.json from job distributor for %s in environment: %s", domain, envKey)

			offChainClient, err := loadOffchainClient(cmd.Context(), domain, envKey, c.lggr)
			if err != nil {
				return err
			}

			// Setup the same filtering as the list command
			domainKey := domain.Key()
			selectors := []*ptypes.Selector{
				{
					Key:   "product",
					Op:    ptypes.SelectorOp_EQ,
					Value: &domainKey,
				},
				{
					Key:   "environment",
					Op:    ptypes.SelectorOp_EQ,
					Value: &envKey,
				},
			}

			// Fetch the nodes from job distributor
			nodes, err := offChainClient.ListNodes(cmd.Context(), &nodev1.ListNodesRequest{
				Filter: &nodev1.ListNodesRequest_Filter{
					Selectors: selectors,
				},
			})
			if err != nil {
				return fmt.Errorf("failed to list nodes: %w", err)
			}

			// Create the nodes structure matching the format in nodes.json
			nodesData := struct {
				Nodes map[string]map[string]string `json:"nodes"`
			}{
				Nodes: make(map[string]map[string]string),
			}

			// Populate the nodes data
			jdNodes := nodes.GetNodes()
			sort.Slice(jdNodes, func(i, j int) bool {
				return jdNodes[i].Name < jdNodes[j].Name
			})
			for _, node := range jdNodes {
				nodesData.Nodes[node.Id] = map[string]string{
					"name": node.Name,
				}
			}

			// Marshal the nodes data
			jsonData, err := json.MarshalIndent(nodesData, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal nodes data: %w", err)
			}

			// In dry-run mode, just print what would be written
			if dryRun {
				c.lggr.Infof("Dry run - would write %d nodes to nodes.json", len(nodesData.Nodes))
				fmt.Println(string(jsonData))

				return nil
			}

			// Get the path to the nodes.json file
			envDir := domain.EnvDir(envKey)
			nodesFilePath := envDir.NodesFilePath()

			// Write the nodes.json file
			err = os.WriteFile(nodesFilePath, jsonData, 0600)
			if err != nil {
				return fmt.Errorf("failed to write nodes.json file: %w", err)
			}

			c.lggr.Infof("Successfully updated nodes.json with %d nodes", len(nodesData.Nodes))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Don't write to nodes.json, just print output")

	return cmd
}

func loadOffchainClient(
	ctx context.Context,
	domain domain.Domain,
	envKey string,
	lggr logger.Logger,
) (foffchain.Client, error) {
	cfg, err := config.LoadEnvConfig(domain, envKey)
	if err != nil {
		return nil, err
	}

	return offchain.LoadOffchainClient(ctx, domain, cfg.Offchain.JobDistributor,
		offchain.WithLogger(lggr),
		offchain.WithCredentials(credentials.GetCredsForEnv(envKey)),
	)
}

type NodeSet struct {
	Nodes []fnode.NodeCfg `json:"nodes" toml:"nodes"`
}
