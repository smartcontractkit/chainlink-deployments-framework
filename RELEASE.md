# Release Process

<!-- TOC -->

- [Release Process](#release-process)
  - [Preparing a Release](#preparing-a-release)
  - [How to Release](#how-to-release)

<!-- TOC -->

### Preparing a Release

After every PR with a changeset is merged, a changesets CI job will create or update a "Version Packages" PR, which contains the release version and information about the changes.

### How to Release

1. Approve or request approval to merge the "Version Packages" PR.
2. Merge the "Version Packages" PR.
3. This will trigger the release workflow, automatically releasing new versions and pushing tags.
4. Root framework releases use `vX.Y.Z` tags.
5. When the `operations-gen` package version is changed, CI also creates `tools/operations-gen/vX.Y.Z` and triggers the operations-gen binary release workflow.
6. Check the [release view](https://github.com/smartcontractkit/chainlink-deployments-framework/releases) to confirm the latest root and operations-gen releases.
