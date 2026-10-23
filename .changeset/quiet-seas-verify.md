---
"chainlink-deployments-framework": patch
---

fix(deps): update Seth to v1.52.0. This version enforces the TLS certificate check for HTTPS RPC connections. To skip the check for a local node, set `SETH_INSECURE_SKIP_VERIFY=true`.
