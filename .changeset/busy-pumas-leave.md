---
"chainlink-deployments-framework": patch
---

fix(engine/test): support multiple mcms deployments

Fix MCMS test helpers to support multiple deployments on same chain

This resolves an issue where MCMS test helpers failed when multiple MCMS 
instances were deployed on the same chain, causing "multiple CallProxy 
addresses found in datastore" errors.
