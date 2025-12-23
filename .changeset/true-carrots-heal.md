---
"chainlink-deployments-framework": minor
---

fix(JD): remove WSRPC field from JDConfig

The WSRPC in JDConfig was never needed as it was never used. Only GRPC field is needed.

