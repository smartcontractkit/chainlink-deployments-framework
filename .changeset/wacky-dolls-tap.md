---
"chainlink-deployments-framework": patch
---

Fix domain scaffolding: the generated `cmd/main.go` hardcoded the `chainlink-deployments` repo in its self import while `go.mod` derived the module path from `{{.repo}}`.
