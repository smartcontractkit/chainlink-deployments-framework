---
"chainlink-deployments-framework": patch
---

fix(catalog): open a DataAccess stream per request instead of one long-lived stream, keeping a single stream for the duration of a transaction
