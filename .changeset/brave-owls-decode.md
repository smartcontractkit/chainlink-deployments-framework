---
"chainlink-deployments-framework": minor
---

Callers can now register an extra Aptos function info lookup with the `analyzer.WithAptosFunctionInfo` proposal context option, for example the CCIP 2.0 `GetFunctionInfo` from `chainlink-ccip-aptos/bindings`. Extra lookups run in order before the `chainlink-aptos` bindings, so proposals that call CCIP 2.0 Aptos modules can be decoded. A lookup that fails or returns entries for the wrong function is skipped. Calls that still cannot be decoded show as a failed method in the report, as before.
