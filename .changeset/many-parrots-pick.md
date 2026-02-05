---
"chainlink-deployments-framework": minor
---

refactor: update stale migration terminology to changeset

Replace legacy "migration" terminology with "changeset" throughout
comments, variable names, and error messages for consistency with
durable pipelines.

Changes include:

- Rename function params: loadMigration → loadChangesets
- Rename variables: migDirPath → dirPath, migration → registry
- Rename test mocks: mockMigrationDS → mockSourceDS
- Update doc comments and error messages
- Remove dead commented-out migration test code
