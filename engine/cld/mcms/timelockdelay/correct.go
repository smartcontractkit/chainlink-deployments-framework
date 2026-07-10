package timelockdelay

import (
	"context"
	"fmt"
	"strings"

	"github.com/smartcontractkit/mcms"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	"github.com/smartcontractkit/chainlink-deployments-framework/pkg/logger"
)

// MinDelayLookup reads on-chain minDelay for one timelock chain entry.
type MinDelayLookup func(
	ctx context.Context,
	proposal *mcms.TimelockProposal,
	chainSelector uint64,
	timelockAddress string,
) (mcmstypes.Duration, error)

// CorrectTimelockDelays updates schedule proposal delays using on-chain minDelay from blockChains.
// Unset or too-low delays are bumped to the max on-chain minDelay across timelock chains when
// minDelay can be read for every timelock chain. An unset delay fails the call if any chain
// cannot be read or on-chain minDelay is zero. An explicitly set delay is left unchanged when
// on-chain minDelay cannot be verified.
func CorrectTimelockDelays(
	ctx context.Context,
	lggr logger.Logger,
	blockChains chain.BlockChains,
	proposals []mcms.TimelockProposal,
) error {
	lookup := func(
		ctx context.Context,
		proposal *mcms.TimelockProposal,
		chainSelector uint64,
		timelockAddress string,
	) (mcmstypes.Duration, error) {
		return readTimelockMinDelay(ctx, blockChains, proposal, chainSelector, timelockAddress)
	}

	return CorrectTimelockDelaysWithLookup(ctx, lggr, proposals, lookup)
}

// CorrectTimelockDelaysWithLookup is like CorrectTimelockDelays but accepts a custom lookup (for tests).
func CorrectTimelockDelaysWithLookup(
	ctx context.Context,
	lggr logger.Logger,
	proposals []mcms.TimelockProposal,
	lookup MinDelayLookup,
) error {
	for i := range proposals {
		if err := correctProposalDelay(ctx, lggr, lookup, &proposals[i]); err != nil {
			return err
		}
	}

	return nil
}

func correctProposalDelay(
	ctx context.Context,
	lggr logger.Logger,
	lookup MinDelayLookup,
	proposal *mcms.TimelockProposal,
) error {
	if proposal == nil || proposal.Action != mcmstypes.TimelockActionSchedule {
		return nil
	}

	unsetDelay := proposal.Delay.Duration <= 0

	if len(proposal.TimelockAddresses) == 0 {
		if unsetDelay {
			return fmt.Errorf("%w: no timelock addresses in proposal", ErrUnsetTimelockDelayUnverified)
		}
		lggr.Warnw("skipping timelock delay correction: no timelock addresses in proposal",
			"action", proposal.Action,
		)

		return nil
	}

	chainDelays, verifyErrors := readChainMinDelaysWithLookup(ctx, lookup, proposal)
	if err := ctx.Err(); err != nil {
		return err
	}

	verifyErrMsg := strings.Join(verifyErrors, "; ")

	if len(verifyErrors) > 0 {
		lggr.Warnw("unable to verify on-chain minDelay for timelock delay correction",
			"errors", verifyErrMsg,
		)
		if unsetDelay {
			return fmt.Errorf("%w: %s", ErrUnsetTimelockDelayUnverified, verifyErrMsg)
		}

		return nil
	}

	maxMinDelay := MaxMinDelay(chainDelays)
	if maxMinDelay.Duration <= 0 {
		if unsetDelay {
			lggr.Warnw("on-chain minDelay is zero on all timelock chains; cannot resolve unset proposal delay")
			return fmt.Errorf("%w: on-chain minDelay is zero on all timelock chains", ErrUnsetTimelockDelayUnverified)
		}

		return nil
	}

	originalDelay := proposal.Delay
	if originalDelay.Duration > 0 && originalDelay.Duration >= maxMinDelay.Duration {
		return nil
	}

	proposal.Delay = maxMinDelay

	lggr.Infow("corrected timelock proposal delay to on-chain minDelay",
		"originalDelay", originalDelay.String(),
		"correctedDelay", maxMinDelay.String(),
	)

	return nil
}
