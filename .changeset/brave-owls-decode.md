---
"chainlink-deployments-framework": minor
---

Callers can now add an Aptos function info lookup with the `analyzer.WithAptosFunctionInfo` proposal context option. The lookup takes a package, module and function name and returns the function info JSON, for example `chainlink-ccip-aptos/bindings.GetFunctionInfo(...).String()`. Lookups run in order before the `chainlink-aptos` bindings, so proposals that call CCIP 2.0 Aptos modules can be decoded. A lookup that returns invalid JSON or no entry for the function is skipped. Calls that still cannot be decoded show as a failed method in the report, as before.
