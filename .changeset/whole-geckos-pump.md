---
"chainlink-deployments-framework": minor
---

feat!: add environment.Build to build an environment from parameters

BREAKING CHANGE: `catalog.LoadCatalog` now takes a domain key, an
environment key and a `cfgenv.CatalogConfig` instead of a domain.Domain
and a *config.Config.
