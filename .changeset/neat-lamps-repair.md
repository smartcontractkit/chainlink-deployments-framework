---
"chainlink-deployments-framework": minor
---

feat: enable strict yaml unmarshalling

When unmarshalling from yaml input for pipelines, if there is a field not defined in the struct, an error will be returned. This helps catch typos and misconfigurations early.
