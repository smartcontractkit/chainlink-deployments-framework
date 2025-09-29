---
"chainlink-deployments-framework": patch
---

fix: preserve large integers in YAML to JSON conversion

Fixes TestSetDurablePipelineInputFromYAML_WithPathResolution by preventing
large integers from being converted to scientific notation during JSON
marshaling, which causes issues when unmarshaling to big.Int.

**Problem:**
- YAML parsing converts large numbers like `2000000000000000000000` to `float64(2e+21)`
- JSON marshaling converts `float64(2e+21)` to scientific notation `"2e+21"`
- big.Int cannot unmarshal scientific notation, causing errors
