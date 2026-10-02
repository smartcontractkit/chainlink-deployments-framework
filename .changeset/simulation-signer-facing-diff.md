---
"chainlink-deployments-framework": minor
---

Add state-view simulation to `execute-fork`: `--simulate-state` forks each chain at a
pinned pre-execution block, runs the proposal's operations against the fork, and emits
a state-diff artifact (`.statediff.json`) recording the before→after of every target
contract's registered views, with read failures quarantined as unreadable-coverage
entries rather than false diffs, and the proposal's pristine signing hash for consumer
verification.
