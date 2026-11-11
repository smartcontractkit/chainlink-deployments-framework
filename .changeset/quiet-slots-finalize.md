---
"chainlink-deployments-framework": patch
---

fix(solana): wait for the finalized commitment in `Chain.Confirm`, so confirmation no longer times out on clusters that do not report the confirmed commitment level
