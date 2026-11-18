---
"chainlink-deployments-framework": patch
---

The proposal analyzer now picks the highest version when one address has several datastore refs of the same type, as a contract upgraded in place does (for example Router 1.6.0 and 2.1.0). Before, the ref read last won, so `mcms analyze-proposal-v2` could decode a call with the old version's decoder and fail on instructions only the new version has. The four copies of the address-mapping code in the analyzer now share one function.
