# Fixture proposals & ref table — provenance

Real committed mainnet proposals (and the datastore ref table needed to resolve
them), copied verbatim so the framework's resolver tests exercise real proposal
shapes without importing the deployments module.

Source repo: `chainlink-deployments` @ `8a2f2990b76cb18454ea8b5c16fc29da4d03079e`
(exported from the main tree on 2026-09-11).

## Proposals (`domains/ccip/mainnet/proposals/`)

| File | Why this one |
|---|---|
| `1786555924138823-ccip-mainnet-promote_candidates_and_set_ocr3-update_chain_configs_merged_proposal.json` | Transactions carry an **empty `contractType`** — the resolver must lean on the datastore for type/version. Chains: 4741433654826277614, 5009297550715157269. |
| `1786654670927127434-ccip-mainnet-set_ocr3_config_mcms_timelock_proposal_0.json` | Sui proposal whose transactions mutate `state_obj` via `additionalFields` — the secondary-target rule. Chain: 17529533435026248318. |
| `1787168710406750-ccip-mainnet-transfer_ownership_raw-deploy_cctp_chains_merged_proposal.json` | EVM multi-target proposal. Chains: 5009297550715157269, 7222032299962346917. |

SHA-256:

```
a5ce92e52bd4c8ded4a5cede6b50dbdc648776a7448be0b74968438fa1d16e5a  1786555924138823-ccip-mainnet-promote_candidates_and_set_ocr3-update_chain_configs_merged_proposal.json
ed6d095e1f043679afbb04aeb7dd0dda504ea1fff2b8f7cc3884068ca4002ce9  1786654670927127434-ccip-mainnet-set_ocr3_config_mcms_timelock_proposal_0.json
9a57150a7983282bc62714a2a986332ad2888597642c5a3281c3e940047ea8c7  1787168710406750-ccip-mainnet-transfer_ownership_raw-deploy_cctp_chains_merged_proposal.json
```

## Refs (`refs/mainnet-fixtures.json`)

Every mainnet datastore address-ref for the four chains above
(`domains/ccip/mainnet/datastore/address_refs.json`), with `labels` stripped
(only `qualifier` is load-bearing: it is part of the datastore ref key, so
dropping it would collide same-typed refs). 452 refs: 365 Ethereum, 39 Sui,
33 selector 7222032299962346917, 15 selector 4741433654826277614.

SHA-256: `596622d5204e2312826fbbe08e325382eb4a2698f4407b9747a9e36f4e316d40`

## Refresh

`refresh.sh` re-copies both from a local deployments checkout and re-verifies
the hashes; it is a no-op without `DEPLOYMENTS_ROOT` set:

```sh
DEPLOYMENTS_ROOT=/path/to/chainlink-deployments ./refresh.sh
```

Hash drift after a refresh is expected (the source files move with mainnet);
update the table above in the same commit.
