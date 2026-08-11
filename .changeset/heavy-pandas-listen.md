---
"chainlink-deployments-framework": patch
---

fix: stop parameter analyzers panicking on calls with no return values. runState indexed the output parameter slice even when annotating an input, so registering any ParameterAnalyzer crashed proposal analysis for nearly every state-changing call
