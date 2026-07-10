package timelockdelay

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/smartcontractkit/mcms"
	mcmschainwrappers "github.com/smartcontractkit/mcms/chainwrappers"
	mcmstypes "github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/chain"
	cldfmcmsadapters "github.com/smartcontractkit/chainlink-deployments-framework/chain/mcms/adapters"
)

// ChainDelay captures on-chain minDelay for one timelock chain.
type ChainDelay struct {
	Selector uint64
	MinDelay mcmstypes.Duration
}

type timelockChainEntry struct {
	selector uint64
	address  string
}

// ReadChainMinDelays reads on-chain minDelay for each timelock address in the proposal.
// Returns per-chain delays and verify error messages for chains that could not be read.
func ReadChainMinDelays(
	ctx context.Context,
	blockChains chain.BlockChains,
	proposal *mcms.TimelockProposal,
) ([]ChainDelay, []string) {
	lookup := func(
		ctx context.Context,
		proposal *mcms.TimelockProposal,
		chainSelector uint64,
		timelockAddress string,
	) (mcmstypes.Duration, error) {
		return readTimelockMinDelay(ctx, blockChains, proposal, chainSelector, timelockAddress)
	}

	return readChainMinDelaysWithLookup(ctx, lookup, proposal)
}

func readChainMinDelaysWithLookup(
	ctx context.Context,
	lookup MinDelayLookup,
	proposal *mcms.TimelockProposal,
) ([]ChainDelay, []string) {
	entries := sortedTimelockEntries(proposal)
	if len(entries) == 0 {
		return nil, nil
	}

	type chainResult struct {
		delay  ChainDelay
		errMsg string
	}

	results := make([]chainResult, len(entries))

	var wg sync.WaitGroup
	wg.Add(len(entries))
	for i, entry := range entries {
		go func(i int, entry timelockChainEntry) {
			defer wg.Done()

			if err := ctx.Err(); err != nil {
				results[i].errMsg = fmt.Sprintf("chain %d (%s): %v", entry.selector, entry.address, err)

				return
			}

			minDelay, err := lookup(ctx, proposal, entry.selector, entry.address)
			if err != nil {
				results[i].errMsg = fmt.Sprintf("chain %d (%s): %v", entry.selector, entry.address, err)

				return
			}
			results[i].delay = ChainDelay{Selector: entry.selector, MinDelay: minDelay}
		}(i, entry)
	}
	wg.Wait()

	chainDelays := make([]ChainDelay, 0, len(entries))
	verifyErrors := make([]string, 0)
	for _, result := range results {
		if result.errMsg != "" {
			verifyErrors = append(verifyErrors, result.errMsg)

			continue
		}
		chainDelays = append(chainDelays, result.delay)
	}

	return chainDelays, verifyErrors
}

// MaxMinDelay returns the largest minDelay across chains.
func MaxMinDelay(chainDelays []ChainDelay) mcmstypes.Duration {
	var maxMinDelay mcmstypes.Duration
	for _, entry := range chainDelays {
		if entry.MinDelay.Duration > maxMinDelay.Duration {
			maxMinDelay = entry.MinDelay
		}
	}

	return maxMinDelay
}

func sortedTimelockEntries(proposal *mcms.TimelockProposal) []timelockChainEntry {
	if proposal == nil || len(proposal.TimelockAddresses) == 0 {
		return nil
	}

	selectors := make([]uint64, 0, len(proposal.TimelockAddresses))
	for selector := range proposal.TimelockAddresses {
		selectors = append(selectors, uint64(selector))
	}
	sort.Slice(selectors, func(i, j int) bool { return selectors[i] < selectors[j] })

	entries := make([]timelockChainEntry, len(selectors))
	for i, selector := range selectors {
		entries[i] = timelockChainEntry{
			selector: selector,
			address:  proposal.TimelockAddresses[mcmstypes.ChainSelector(selector)],
		}
	}

	return entries
}

func readTimelockMinDelay(
	ctx context.Context,
	blockChains chain.BlockChains,
	proposal *mcms.TimelockProposal,
	chainSelector uint64,
	timelockAddress string,
) (mcmstypes.Duration, error) {
	chainAccessor := cldfmcmsadapters.Wrap(blockChains)
	metadata := chainMetadataForInspector(proposal, chainSelector)

	inspector, err := mcmschainwrappers.BuildTimelockInspector(
		&chainAccessor,
		mcmstypes.ChainSelector(chainSelector),
		metadata,
	)
	if err != nil {
		return mcmstypes.Duration{}, err
	}

	minDelaySec, err := inspector.GetMinDelay(ctx, timelockAddress)
	if err != nil {
		return mcmstypes.Duration{}, err
	}

	return durationFromSeconds(minDelaySec)
}

func chainMetadataForInspector(proposal *mcms.TimelockProposal, chainSelector uint64) mcmstypes.ChainMetadata {
	if proposal == nil || len(proposal.ChainMetadata) == 0 {
		return mcmstypes.ChainMetadata{}
	}

	metadata, ok := proposal.ChainMetadata[mcmstypes.ChainSelector(chainSelector)]
	if !ok {
		return mcmstypes.ChainMetadata{}
	}

	return metadata
}

func durationFromSeconds(seconds uint64) (mcmstypes.Duration, error) {
	maxSeconds := uint64(math.MaxInt64 / int64(time.Second))
	if seconds > maxSeconds {
		return mcmstypes.Duration{}, fmt.Errorf("min delay %d seconds exceeds representable duration", seconds)
	}

	return mcmstypes.NewDuration(time.Duration(seconds) * time.Second), nil
}
