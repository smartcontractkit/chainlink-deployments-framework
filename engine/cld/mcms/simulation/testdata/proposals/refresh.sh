#!/usr/bin/env sh
# Re-copies the fixture proposals and the ref table from a local
# chainlink-deployments checkout and re-prints their hashes. No-op unless
# DEPLOYMENTS_ROOT points at a checkout.
set -eu

if [ -z "${DEPLOYMENTS_ROOT:-}" ]; then
	echo "DEPLOYMENTS_ROOT not set; nothing to do."
	exit 0
fi

ROOT="$(cd "$(dirname "$0")" && pwd)"
PROPOSALS="$DEPLOYMENTS_ROOT/domains/ccip/mainnet/proposals"
REFS="$DEPLOYMENTS_ROOT/domains/ccip/mainnet/datastore/address_refs.json"

for f in \
	1786555924138823-ccip-mainnet-promote_candidates_and_set_ocr3-update_chain_configs_merged_proposal.json \
	1786654670927127434-ccip-mainnet-set_ocr3_config_mcms_timelock_proposal_0.json \
	1787168710406750-ccip-mainnet-transfer_ownership_raw-deploy_cctp_chains_merged_proposal.json; do
	cp "$PROPOSALS/$f" "$ROOT/$f"
done

jq '[
  .[] |
  select(
    .chainSelector == 5009297550715157269 or
    .chainSelector == 4741433654826277614 or
    .chainSelector == 17529533435026248318 or
    .chainSelector == 7222032299962346917
  ) | del(.labels)
]' "$REFS" > "$ROOT/../refs/mainnet-fixtures.json"

echo "Refreshed. New hashes (update README.md in the same commit):"
sha256sum "$ROOT"/*.json "$ROOT/../refs/mainnet-fixtures.json"
