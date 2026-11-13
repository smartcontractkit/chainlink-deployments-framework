---
"chainlink-deployments-framework": minor
---

Add opt-in RPC discovery: set `rpc_manifest_url` in `domain.yaml` and networks with an empty `rpcs:` list are filled from the RPC manifest. YAML-defined RPCs always take precedence, and nothing is fetched when the setting is unset. `CLD_RPC_MANIFEST_URL` overrides the domain setting (e.g. for a local file).
