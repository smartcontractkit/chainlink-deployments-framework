---
"chainlink-deployments-framework": minor
---

The `mcms` command group (`analyze-proposal-v2`, `convert-upf` and the deprecated `analyze-proposal`) can now build its environment from a BuildParams YAML document instead of reading a `domains/<domain>/` directory. Supply it with the `MCMS_ENV_BUILD_PARAMS` environment variable or the `--env-build-params` flag, the same document and names the `mcms` CLI uses. Without either, behavior is unchanged.

Adds the `engine/cld/environment/envparams` package, which resolves the flag and variable, and `Deps.EnvironmentBuilder` and `Deps.Getenv` on the mcms commands.
