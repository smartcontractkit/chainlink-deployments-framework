---
"chainlink-deployments-framework": minor
---

feat: introduce template-input command for generating YAML input
    
This commit introduces a new template-input command that generates YAML input templates from Go struct types for durable pipeline changesets. The command uses reflection to analyze changeset input types and produces well-formatted YAML templates with type comments to guide users in creating valid input files.
