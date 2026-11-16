---
"chainlink-deployments-framework": minor
---

The `mcms error-decode-evm` command now gives output that a program can read, and it can run without a domain directory. Use `--format json` to get one JSON object on stdout with the raw and decoded `revert_reason` and `underlying_reason`. Use `--error-file -` to read the error from stdin. Use `--env-build-params` or `MCMS_ENV_BUILD_PARAMS` to build the environment from a BuildParams document, as the analyze and convert commands do.

The command also decodes custom errors that have arguments, for example errors from the CCIP router. Before, it read the first 4 bytes of the arguments as the selector, and it showed a wrong name such as `custom error 0x00000000`.
