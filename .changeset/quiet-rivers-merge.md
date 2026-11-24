---
"chainlink-deployments-framework": minor
---

Add opt-in RPC discovery: set `rpc_manifest_url` (an HTTP(S) URL) in `domain.yaml` and networks with an empty `rpcs:` list are filled from the RPC manifest. YAML-defined RPCs always take precedence, and nothing is fetched when the setting is unset. A network can set `rpcs_mode: append` to keep its own `rpcs` first and add the manifest's provider RPCs after them; the default (`fallback`) uses yaml `rpcs` as-is. Provider RPC URLs are built on the manifest URL's host.
