---
"chainlink-deployments-framework": minor
---

- Removed legacy migration CLI commands that have been superseded by durable pipelines: - `migration run` - Use `durable-pipeline run` instead - `migration list` - Use `durable-pipeline list` instead - `migration latest` - No longer supported - `migration address-book` - Use top-level `address-book` command instead - `migration datastore` - Use top-level `datastore` command instead The following files have been removed: - `engine/cld/legacy/cli/commands/migration.go` - `engine/cld/legacy/cli/commands/migration_test.go` - `engine/cld/legacy/cli/commands/migration_helper.go`
