---
"chainlink-deployments-framework": minor
---

Adds a new test engine task to sign and execute all pending proposals

A new test engine runtime task has been added to improve the experience
of signing and executing MCMS proposals. This new task will sign and
execute all pending proposals that previous ChangesetTasks have generated.

```
signingKey, _ := crypto.GenerateKey() // Use your actual MCMS signing key here instead
runtime.Exec(
    SignAndExecuteProposalsTask([]*ecdsa.PrivateKey{signingKey},
)
```
