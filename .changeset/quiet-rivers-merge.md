---
"chainlink-deployments-framework": minor
---

Add opt-in RPC discovery: when `CLD_RPC_MANIFEST_URL` is set, networks with an empty `rpcs:` list are filled from the RPC manifest. YAML-defined RPCs always take precedence, and nothing is fetched when the variable is unset.
