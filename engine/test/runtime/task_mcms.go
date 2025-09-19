package runtime

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/segmentio/ksuid"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	fdeployment "github.com/smartcontractkit/chainlink-deployments-framework/deployment"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/test/internal/mcmsutils"
)

// SignAllProposalsTask creates a new executable task that signs all proposals in the state.
func SignAllProposalsTask(privateKey *ecdsa.PrivateKey) signAllProposalsTask {
	return signAllProposalsTask{
		id:         ksuid.New().String(),
		privateKey: privateKey,
	}
}

type signAllProposalsTask struct {
	id         string // Unique identifier for this task
	privateKey *ecdsa.PrivateKey
}

func (t signAllProposalsTask) ID() string {
	return t.id
}

func (t signAllProposalsTask) Run(e fdeployment.Environment, state *State) error {
	ctx := e.GetContext()

	// Create a new signer
	signer, err := mcmsutils.NewSigner(e)
	if err != nil {
		return fmt.Errorf("new signer: %w", err)
	}

	for _, props := range state.Proposals {
		for i, prop := range props {
			kind, err := proposalKind(prop)
			if err != nil {
				return fmt.Errorf("proposal kind: %w", err)
			}

			var proposalJSON string
			switch kind {
			case mcmstypes.KindProposal:
				p, err := mcmsutils.DecodeProposal(prop)
				if err != nil {
					return fmt.Errorf("new proposal: %w", err)
				}

				if err = signer.SignMCMS(ctx, p, t.privateKey, true); err != nil {
					return fmt.Errorf("new proposal: %w", err)
				}

				proposalJSON, err = mcmsutils.EncodeProposal(p)
				if err != nil {
					return fmt.Errorf("encode proposal: %w", err)
				}
			case mcmstypes.KindTimelockProposal:
				p, err := mcmsutils.DecodeTimelockProposal(prop)
				if err != nil {
					return fmt.Errorf("new proposal: %w", err)
				}

				if err = signer.SignTimelock(ctx, p, t.privateKey, true); err != nil {
					return fmt.Errorf("new timelock proposal: %w", err)
				}

				proposalJSON, err = mcmsutils.EncodeTimelockProposal(p)
				if err != nil {
					return fmt.Errorf("encode timelock proposal: %w", err)
				}
			default:
				return fmt.Errorf("unsupported proposal kind: %s", kind)
			}

			props[i] = proposalJSON
		}
	}

	return nil
}

// proposalKind extracts the kind of a proposal from a JSON proposal.
func proposalKind(p string) (mcmstypes.ProposalKind, error) {
	type proposal struct {
		Kind mcmstypes.ProposalKind `json:"kind"`
	}

	var prop proposal
	if err := json.Unmarshal([]byte(p), &prop); err != nil {
		return "", err
	}

	return prop.Kind, nil
}
