// Package simulation resolves the per-chain address scope a proposal's state
// diff covers: the targets to diff, and the read-only context that dependent
// views need to resolve correctly.
package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/smartcontractkit/mcms"
	"github.com/smartcontractkit/mcms/types"

	"github.com/smartcontractkit/chainlink-deployments-framework/datastore"
	"github.com/smartcontractkit/chainlink-deployments-framework/engine/cld/mcms/simulation/statediff"
)

// Scope is the resolved address scope for one proposal simulation on one chain.
type Scope struct {
	ChainSelector uint64
	// Targets are the addresses whose state changes are diffed.
	Targets []datastore.AddressRef
	// Context are read-only addresses included so dependent views resolve:
	// routers and tokens feed the v1.6 FeeQuoter and OffRamp views. Context
	// state is never diffed.
	Context []datastore.AddressRef
}

// ResolveScope resolves the diff targets and view context for chainSelector
// from the proposal and the datastore. The datastore is authoritative for
// contract type and version (proposal transactions frequently carry an empty
// contractType and never a version). Target addresses absent from the
// datastore are reported Unreadable, not dropped: an unknown address is itself
// a signal worth surfacing.
func ResolveScope(
	_ context.Context,
	refs datastore.AddressRefStore,
	proposal *mcms.TimelockProposal,
	chainSelector types.ChainSelector,
) (Scope, []statediff.Unreadable, error) {
	meta, ok := proposal.ChainMetadata[chainSelector]
	if !ok {
		return Scope{}, nil, fmt.Errorf("proposal has no chain metadata for selector %d", chainSelector)
	}

	byAddr := make(map[string]datastore.AddressRef)
	for _, r := range refs.Filter(datastore.AddressRefByChainSelector(uint64(chainSelector))) {
		byAddr[strings.ToLower(r.Address)] = r
	}

	// Collect target addresses: the transaction's to-address plus
	// family-specific secondary targets (on Sui the mutated object is
	// state_obj, which is not always the to-address).
	targetAddrs := []string{}
	addTarget := func(addr string) {
		addr = strings.ToLower(strings.TrimSpace(addr))
		if addr != "" && !slices.Contains(targetAddrs, addr) {
			targetAddrs = append(targetAddrs, addr)
		}
	}
	for _, op := range proposal.Operations {
		if uint64(op.ChainSelector) != uint64(chainSelector) {
			continue
		}
		for _, tx := range op.Transactions {
			addTarget(tx.To)
			for _, secondary := range secondaryTargets(tx.AdditionalFields) {
				addTarget(secondary)
			}
		}
	}

	scope := Scope{ChainSelector: uint64(chainSelector)}
	unreadable := []statediff.Unreadable{}
	for _, addr := range targetAddrs {
		if ref, ok := byAddr[addr]; ok {
			scope.Targets = append(scope.Targets, ref)
		} else {
			unreadable = append(unreadable, statediff.Unreadable{
				ChainSelector: uint64(chainSelector),
				Address:       addr,
				Reason:        statediff.ReasonUnknownAddress,
			})
		}
	}

	// Context: the chain's MCM and timelock (their bookkeeping noise is what
	// the differ's ignore list is calibrated against), plus every router and
	// token ref on the chain.
	seen := map[string]struct{}{}
	addContext := func(ref datastore.AddressRef) {
		key := strings.ToLower(ref.Address)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		scope.Context = append(scope.Context, ref)
	}
	for _, addr := range []string{meta.MCMAddress, proposal.TimelockAddresses[chainSelector]} {
		if ref, ok := byAddr[strings.ToLower(addr)]; ok {
			addContext(ref)
		}
	}
	contextAddrs := make([]string, 0, len(byAddr))
	for addr, ref := range byAddr {
		if strings.Contains(string(ref.Type), "Router") || strings.Contains(string(ref.Type), "Token") {
			contextAddrs = append(contextAddrs, addr)
		}
	}
	slices.Sort(contextAddrs)
	for _, addr := range contextAddrs {
		addContext(byAddr[addr])
	}

	return scope, unreadable, nil
}

// secondaryTargets extracts family-specific secondary diff targets from a
// transaction's additionalFields. Currently only Sui's state_obj, which can be
// a single object id or a list.
func secondaryTargets(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil
	}
	switch v := fields["state_obj"].(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}

	return nil
}

// AddressRefs renders a scope's refs as a compact string for logs.
func (s Scope) AddressRefs() string {
	parts := make([]string, 0, len(s.Targets)+len(s.Context))
	for _, r := range s.Targets {
		parts = append(parts, "T:"+r.Address)
	}
	for _, r := range s.Context {
		parts = append(parts, "C:"+r.Address)
	}

	return strconv.Itoa(len(parts)) + " refs"
}
