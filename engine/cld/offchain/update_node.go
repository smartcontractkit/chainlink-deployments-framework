package offchain

import (
	"context"
	"errors"
	"fmt"

	nodev1 "github.com/smartcontractkit/chainlink-protos/job-distributor/v1/node"

	cldf_offchain "github.com/smartcontractkit/chainlink-deployments-framework/offchain"
	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/node"

	"github.com/smartcontractkit/chainlink-protos/job-distributor/v1/shared/ptypes"
)

// UpdateNodeRequest is the request to update a node using JD
type UpdateNodeRequest struct {
	// Cfg is the configuration for the used to update node
	Cfg node.NodeCfg
	// keyCfg is the configuration for how to find the node to update
	keyCfg nodeKeyCfg
}

// NodeFinderCfg is the configuration for how to find the node to update
type NodeFinderCfg struct {
	KeyType   NodeKey // type of key to search for
	LabelName *string // name of label to search for when using NodeKey_Label
}

func (c NodeFinderCfg) Validate() error {
	if c.KeyType == NodeKey_Label && c.LabelName == nil {
		return errors.New("label name is required for label search")
	}

	return nil
}

// NewUpdateNodeRequest creates a new UpdateNodeRequest
func NewUpdateNodeRequest(cfg node.NodeCfg, f NodeFinderCfg) (*UpdateNodeRequest, error) {
	if err := f.Validate(); err != nil {
		return nil, fmt.Errorf("invalid node finder config: %w", err)
	}
	kc, err := newNodeKeyCfg(cfg, f)
	if err != nil {
		return nil, err
	}

	return &UpdateNodeRequest{
		Cfg:    cfg,
		keyCfg: kc,
	}, nil
}

// Labels returns the labels for the node, containing the p2p_id, nop, admin_addr and all tags.
func (r *UpdateNodeRequest) Labels() []*ptypes.Label {
	raw := r.Cfg.Labels()

	out := make([]*ptypes.Label, 0, len(raw))
	for k, v := range raw {
		out = append(out, &ptypes.Label{
			Key:   k,
			Value: &v,
		})
	}

	return out
}

// NodeKeyCriteria returns the node key criteria
func (r *UpdateNodeRequest) NodeKeyCriteria() string {
	return r.keyCfg.String()
}

// UpdateNodesRequest is the request to update multiple nodes using JD
type UpdateNodesRequest struct {
	Requests []*UpdateNodeRequest
}

// UpdateNodes updates the nodes with the given configurations.
func UpdateNodes(ctx context.Context, client cldf_offchain.Client, req UpdateNodesRequest) error {
	if len(req.Requests) == 0 {
		return nil
	}
	resp, err := client.ListNodes(ctx, &nodev1.ListNodesRequest{})
	if err != nil {
		return err
	}
	for _, r := range req.Requests {
		node, err := getNode(resp, r.keyCfg.keyType, r.keyCfg.value, r.keyCfg.labelKey)
		if err != nil {
			return fmt.Errorf("failed to get node %s with value %s (label=%v): %w", r.
				keyCfg.keyType, r.keyCfg.value, r.keyCfg.labelKey, err)
		}
		_, err = client.UpdateNode(ctx, &nodev1.UpdateNodeRequest{
			Id:        node.GetId(),
			Name:      r.Cfg.Name,
			PublicKey: r.Cfg.CSAKey,
			Labels:    r.Labels(),
		})
		if err != nil {
			return fmt.Errorf("failed to update node %s name %s csakey %s: %w", node.GetId(), r.Cfg.Name, r.Cfg.CSAKey, err)
		}
	}

	return nil
}

// getNode gets a node from the list of nodes based on the key type and key value.
// the key is only used for label searches and it is required for label searches.
func getNode(resp *nodev1.ListNodesResponse, keyType NodeKey, value string, labelKey *string) (*nodev1.Node, error) {
	switch keyType {
	case NodeKey_ID:
		return getNodeByID(resp, value)
	case NodeKey_CSAKey:
		return getNodeByCSAKey(resp, value)
	case NodeKey_Name:
		return getNodeByName(resp, value)
	case NodeKey_Label:
		if labelKey == nil {
			return nil, errors.New("no key provided for label search")
		}

		return getNodeByLabel(resp, *labelKey, value)
	default:
		return nil, fmt.Errorf("unknown key type %s", keyType)
	}
}

func getNodeByID(resp *nodev1.ListNodesResponse, id string) (*nodev1.Node, error) {
	for _, node := range resp.GetNodes() {
		if node.GetId() == id {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no node with id %s found", id)
}

func getNodeByCSAKey(resp *nodev1.ListNodesResponse, csaKey string) (*nodev1.Node, error) {
	for _, node := range resp.GetNodes() {
		if node.GetPublicKey() == csaKey {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no node with csa key %s found", csaKey)
}

func getNodeByName(resp *nodev1.ListNodesResponse, name string) (*nodev1.Node, error) {
	for _, node := range resp.GetNodes() {
		if node.GetName() == name {
			return node, nil
		}
	}

	return nil, fmt.Errorf("no node with name %s found", name)
}

func getNodeByLabel(resp *nodev1.ListNodesResponse, key, value string) (*nodev1.Node, error) {
	for _, node := range resp.GetNodes() {
		for _, label := range node.GetLabels() {
			if label.GetKey() == key && label.GetValue() == value {
				return node, nil
			}
		}
	}

	return nil, fmt.Errorf("no node with label %s=%s found", key, value)
}

type nodeKeyCfg struct {
	keyType  NodeKey // type of key to search for
	value    string  // value to search for
	labelKey *string // key to search for in labels when using NodeKey_Label
}

func (c nodeKeyCfg) String() string {
	v := c.value
	if c.labelKey != nil {
		v = fmt.Sprintf("%s=%s", *c.labelKey, c.value)
	}

	return fmt.Sprintf("key-type=%s, value=%s", c.keyType, v)
}

func newNodeKeyCfg(n node.NodeCfg, c NodeFinderCfg) (nodeKeyCfg, error) {
	out := nodeKeyCfg{
		keyType: c.KeyType,
	}
	switch c.KeyType {
	case NodeKey_CSAKey:
		out.value = n.CSAKey
	case NodeKey_Name:
		out.value = n.Name
	case NodeKey_Label:
		out.value = n.Labels()[*c.LabelName]
		out.labelKey = c.LabelName
	case NodeKey_ID:
		return nodeKeyCfg{}, errors.New("id key type is not supported")
	default:
		return nodeKeyCfg{}, fmt.Errorf("unknown key type %s", c.KeyType)
	}

	return out, nil
}
