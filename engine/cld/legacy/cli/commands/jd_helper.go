package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	chainsel "github.com/smartcontractkit/chain-selectors"
	jobv1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/job"
	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	foffchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
)

// listJobsByNodeId fetches jobs for a slice of node IDs and returns a map of job slices for each node ID.
func listJobsByNodeId(
	ctx context.Context,
	offChainClient foffchain.Client,
	nodeIds []string,
) (map[string][]*jobv1.Job, error) {
	jobs, err := offChainClient.ListJobs(ctx, &jobv1.ListJobsRequest{
		Filter: &jobv1.ListJobsRequest_Filter{
			NodeIds: nodeIds,
		},
	})
	if err != nil {
		return nil, err
	}

	return toJobsByNodeId(jobs.GetJobs()), nil
}

// toJobsByNodeId transforms a slice of jobs into a map of jobs keyed by the node ID.
func toJobsByNodeId(jobs []*jobv1.Job) map[string][]*jobv1.Job {
	nid2job := make(map[string][]*jobv1.Job)
	for _, j := range jobs {
		nid2job[j.NodeId] = append(nid2job[j.NodeId], j)
	}

	return nid2job
}

// listProposalsByNodeId fetches the proposals for a list of jobs for a node and maps these proposals
// to a set of node Id.
func listProposalsByNodeId(
	ctx context.Context,
	offChainClient foffchain.Client,
	nodeIdToJobs map[string][]*jobv1.Job,
) (map[string][]*jobv1.Proposal, error) {
	nid2proposal := make(map[string][]*jobv1.Proposal)
	for nid, jobs := range nodeIdToJobs {
		ps, err := offChainClient.ListProposals(ctx, &jobv1.ListProposalsRequest{
			Filter: &jobv1.ListProposalsRequest_Filter{
				JobIds: toJobIDs(jobs),
			},
		})
		if err != nil {
			return nil, err
		}

		nid2proposal[nid] = ps.GetProposals()
	}

	return nid2proposal, nil
}

// toNodeIDs maps nodes into slice of IDs.
func toNodeIDs(nodes []*nodev1.Node) []string {
	nodeIDs := make([]string, 0, len(nodes))
	for _, ni := range nodes {
		nodeIDs = append(nodeIDs, ni.Id)
	}

	return nodeIDs
}

// toJobIDs maps job slice into slice of IDs.
func toJobIDs(jobs []*jobv1.Job) []string {
	ids := make([]string, 0, len(jobs))
	for _, j := range jobs {
		ids = append(ids, j.Id)
	}

	return ids
}

// toNodeViews maps a slice of nodes into a view that includes the jobs and their proposals.
func toNodeViews(
	ns []*nodev1.Node,
	nidToProposals map[string][]*jobv1.Proposal,
	nidToJobs map[string][]*jobv1.Job,
) []NodeView {
	nvs := make([]NodeView, 0, len(ns))
	for _, n := range ns {
		jobsView := toJobsView(n, nidToProposals, nidToJobs)
		nvs = append(nvs, NodeView{
			Node:      n,
			CreatedAt: n.CreatedAt.AsTime().String(),
			UpdatedAt: n.UpdatedAt.AsTime().String(),
			Jobs:      jobsView,
		})
	}

	return nvs
}

// toJobsView maps a slice of nodes into a map of job UUID to slice of proposals.
func toJobsView(
	n *nodev1.Node,
	nidToProposals map[string][]*jobv1.Proposal,
	nidToJobs map[string][]*jobv1.Job,
) map[string][]ProposalView {
	jobs := nidToJobs[n.GetId()]
	pvsByJobID := toProposalsByJobID(nidToProposals[n.Id])
	jobsView := make(map[string][]ProposalView)
	for _, j := range jobs {
		jobsView[j.GetUuid()] = pvsByJobID[j.GetId()]
	}

	return jobsView
}

// toProposalsByJobID maps job ID to a slice of proposal views.
func toProposalsByJobID(proposals []*jobv1.Proposal) map[string][]ProposalView {
	proposalsByJobID := make(map[string][]ProposalView, 0)
	for _, p := range proposals {
		proposalsByJobID[p.JobId] = append(proposalsByJobID[p.JobId], toProposalView(p))
	}

	return proposalsByJobID
}

// toProposalViews transforms a Proposal to a ProposalView
func toProposalView(p *jobv1.Proposal) ProposalView {
	return ProposalView{
		ID:                 p.Id,
		JobID:              p.GetJobId(),
		Status:             p.Status.String(),
		Revision:           p.GetRevision(),
		DeliveryStatus:     p.DeliveryStatus.String(),
		CreatedAt:          p.CreatedAt.AsTime().String(),
		UpdatedAt:          p.UpdatedAt.AsTime().String(),
		AckedAt:            p.AckedAt.AsTime().String(),
		ResponseReceivedAt: p.ResponseReceivedAt.AsTime().String(),
	}
}

func writeNodeTable(nodes []*nodev1.Node) {
	for _, node := range nodes {
		labelsString := &strings.Builder{}
		labelsTable := tablewriter.NewWriter(labelsString)
		labels := make([][]string, 0, len(node.Labels))
		for _, label := range node.Labels {
			labels = append(labels, []string{label.Key, *label.Value})
		}
		labelsTable.SetBorders(tablewriter.Border{
			Left:   false,
			Right:  false,
			Top:    true,
			Bottom: true,
		})
		labelsTable.AppendBulk(labels)
		labelsTable.Render()

		data := [][]string{
			{"ID", node.Id},
			{"Name", node.Name},
			{"CSA", node.PublicKey},
		}
		if node.WorkflowKey != nil {
			data = append(data, []string{"WorkflowKey", *node.WorkflowKey})
		}

		if len(node.P2PKeyBundles) > 0 {
			p2pBuilder := &strings.Builder{}
			p2pTable := tablewriter.NewWriter(p2pBuilder)
			// Pre-allocate capacity for 2 rows per P2PKeyBundle (Peer ID and Public Key).
			// If the number of rows per bundle changes, update this multiplier accordingly.
			p2pData := make([][]string, 0, len(node.P2PKeyBundles)*2)
			for _, p2p := range node.P2PKeyBundles {
				p2pData = append(p2pData, []string{"Peer ID", p2p.PeerId})
				p2pData = append(p2pData, []string{"Public Key", p2p.PublicKey})
			}
			p2pTable.SetBorders(tablewriter.Border{
				Left:   false,
				Right:  false,
				Top:    true,
				Bottom: true,
			})
			p2pTable.AppendBulk(p2pData)
			p2pTable.Render()
			data = append(data, []string{"P2P Key Bundles", p2pBuilder.String()})
		}

		data = append(data,
			[]string{"Enabled", strconv.FormatBool(node.IsEnabled)},
			[]string{"Connected", strconv.FormatBool(node.IsConnected)},
			[]string{"Labels", labelsString.String()},
			[]string{"Version", node.Version},
			[]string{"Created at", node.CreatedAt.AsTime().Format(time.RFC3339)},
			[]string{"Updated at", node.UpdatedAt.AsTime().Format(time.RFC3339)},
		)
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoWrapText(false)
		table.AppendBulk(data)
		table.Render()
	}
}

func writeChainConfigTable(configsByNode map[string][]*nodev1.ChainConfig) {
	for nodeID, configs := range configsByNode {
		fmt.Printf("Node ID: %v\n", nodeID)
		for _, config := range configs {
			var family string
			switch config.Chain.Type {
			case nodev1.ChainType_CHAIN_TYPE_EVM:
				family = chainsel.FamilyEVM
			case nodev1.ChainType_CHAIN_TYPE_APTOS:
				family = chainsel.FamilyAptos
			case nodev1.ChainType_CHAIN_TYPE_SOLANA:
				family = chainsel.FamilySolana
			case nodev1.ChainType_CHAIN_TYPE_STARKNET:
				family = chainsel.FamilyStarknet
			case nodev1.ChainType_CHAIN_TYPE_TRON:
				family = chainsel.FamilyTron
			case nodev1.ChainType_CHAIN_TYPE_TON:
				family = chainsel.FamilyTon
			case nodev1.ChainType_CHAIN_TYPE_SUI:
				family = chainsel.FamilySui
			case nodev1.ChainType_CHAIN_TYPE_UNSPECIFIED:
				panic("chain type must be specified")
			default:
				panic(fmt.Sprintf("unsupported chain type %s", config.Chain.Type))
			}

			details, err := chainsel.GetChainDetailsByChainIDAndFamily(config.Chain.Id, family)
			if err != nil {
				panic(err)
			}

			data := [][]string{
				{"Chain", fmt.Sprintf("%s (network=%s chainID=%s)", details.ChainName, config.Chain.Type.String(), config.Chain.Id)},
				{"Admin Address", config.AdminAddress},
				{"Account Address", config.AccountAddress},
			}
			if config.AccountAddressPublicKey != nil {
				data = append(data, []string{"Account PublicKey", *config.AccountAddressPublicKey})
			}
			if config.FluxMonitorConfig != nil && config.FluxMonitorConfig.Enabled {
				data = append(data, []string{"FluxMonitor", fmt.Sprintf("%+v", config.FluxMonitorConfig)})
			}
			if config.Ocr1Config != nil && config.Ocr1Config.Enabled {
				data = append(data, []string{"OCR1", fmt.Sprintf("%+v", config.Ocr1Config)})
			}
			if config.Ocr2Config != nil && config.Ocr2Config.Enabled {
				ocr2String := &strings.Builder{}
				ocr2Table := tablewriter.NewWriter(ocr2String)
				ocr2Data := [][]string{
					{"IsBootstrap", strconv.FormatBool(config.Ocr2Config.IsBootstrap)},
					{"Multiaddr", config.Ocr2Config.Multiaddr},
					{"ForwarderAddress", *config.Ocr2Config.ForwarderAddress},
					{"Plugins", fmt.Sprintf("%+v", config.Ocr2Config.Plugins)},
					// p2p
					{"Peer ID", config.Ocr2Config.P2PKeyBundle.PeerId},
					{"P2P Public Key", config.Ocr2Config.P2PKeyBundle.PublicKey},
					// ocr2
					{"Key Bundle ID", config.Ocr2Config.OcrKeyBundle.BundleId},
					{"Config Public Key", config.Ocr2Config.OcrKeyBundle.ConfigPublicKey},
					{"Offchain Public Key", config.Ocr2Config.OcrKeyBundle.OffchainPublicKey},
					{"Onchain Signing Key", config.Ocr2Config.OcrKeyBundle.OnchainSigningAddress},
				}
				ocr2Table.SetBorders(tablewriter.Border{
					Left:   false,
					Right:  false,
					Top:    true,
					Bottom: true,
				})
				ocr2Table.AppendBulk(ocr2Data)
				ocr2Table.Render()
				data = append(data, []string{"OCR2", ocr2String.String()})
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetAutoWrapText(false)
			table.AppendBulk(data)
			table.Render()
		}
		fmt.Println()
	}
}

func parseLabelsArg(l []string) (map[string]string, error) {
	labels := map[string]string{}
	for _, label := range l {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format '%v'", label)
		}
		labels[parts[0]] = parts[1]
	}

	return labels, nil
}
