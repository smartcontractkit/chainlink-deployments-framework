# Release Versioning Guide

## Overview

The `chainlink-deployments-framework` repository follows [Semantic Versioning (SemVer)](https://semver.org/) principles and uses [Changesets](https://github.com/changesets/changesets) to manage versioning and release notes. This guide provides guidelines for contributors to ensure consistent and meaningful version management.

---

## Semantic Versioning Principles

Following the [SemVer specification](https://semver.org/), our version numbers follow the format `MAJOR.MINOR.PATCH`:

### MAJOR Version (`X.0.0`)

**When to increment**: Making incompatible API changes

**Examples**:

- Removing public functions, methods, or types
- Changing function signatures (parameters, return types)
- Removing or renaming public fields in structs
- Changing behavior that breaks existing integrations
- Removing support for deprecated features

### MINOR Version (`X.Y.0`)

**When to increment**: Adding functionality in a backward-compatible manner

**Examples**:

- Adding new public functions, methods, or types
- Adding new optional parameters using function options pattern
- Adding new fields to structs (that don't break existing usage)
- Deprecating functionality (while maintaining backward compatibility)
- Performance improvements that don't change behavior

### PATCH Version (`X.Y.Z`)

**When to increment**: Making backward-compatible bug fixes

**Examples**:

- Fixing incorrect behavior without changing the API
- Internal refactoring that doesn't affect public interfaces
- Documentation updates
- Dependency updates that don't affect the public API
- Security fixes that maintain API compatibility

---

## Changeset Workflow

### 1. Creating a Changeset

When making changes that affect the public API or behavior, create a changeset:

```shell
pnpm changeset
```

This will prompt you to:

1. Select which packages are affected
2. Choose the type of change (patch, minor, major)
3. Write a summary of the changes

### 2. Changeset File Structure

Changesets are stored as markdown files in the `.changeset` directory (managed automatically):

```markdown
---
"chainlink-deployments-framework": major
---

feat: support feature A

<description here>
```

### 3. Release Process

The release process is automated via CI.

---

## Version Increment Guidelines

### Decision Tree

```
Is this a breaking change to the public API?
├─ YES → MAJOR version
└─ NO → Is this adding new functionality?
    ├─ YES → MINOR version
    └─ NO → PATCH version
```

### What Constitutes a Breaking Change?

**✅ Breaking Changes (MAJOR)**:

- Removing exported functions, types, or constants
- Changing function signatures
- Changing struct field names or types
- Removing or changing behavior of existing features
- Changing default values that affect behavior
- Removing deprecated features

**❌ Not Breaking Changes**:

- Adding new exported functions or types
- Adding new optional fields to structs
- Adding new methods to interfaces (with default implementations)
- Internal refactoring without API changes
- Bug fixes that restore intended behavior
- Performance improvements

---

## Breaking Changes and Guidelines

### When Making Breaking Changes

- **Justify the Breaking Change**: Ensure it's necessary and provides significant value
- **Create Comprehensive Migration Guide**: Document every step users need to take
- **Provide Examples**: Show before/after code examples
- **Consider Deprecation Path**: When possible, deprecate first, then remove in next major version

### Don't Fear Major Versions

- **Major versions are normal**: They communicate important changes to users
- **Better than technical debt**: Clean breaks are better than maintaining bad APIs
- **Plan major versions**: Group breaking changes when possible

---

## Examples

### Example 1: Adding a New Feature (MINOR)

```markdown
---
"chainlink-deployments-framework": minor
---

feat: add support for custom retry strategies

Added new RetryStrategy interface and implementations for exponential backoff and linear retry patterns. This allows users to customize how operations are retried without breaking existing behavior.

- New `RetryStrategy` interface
- `ExponentialBackoffStrategy` and `LinearRetryStrategy` implementations
- `WithRetryStrategy()` option for configuring custom retry behavior
- Existing retry behavior remains unchanged (backward compatible)
```

### Example 2: Bug Fix (PATCH)

```markdown
---
"chainlink-deployments-framework": patch
---

fix: correct timeout calculation in HTTP client

Fixed a bug where HTTP client timeouts were being calculated incorrectly, causing requests to timeout prematurely. This fix restores the intended behavior without changing the public API.

- Fixed timeout calculation logic
- Added unit tests for timeout scenarios
- No API changes required
```

### Example 3: Breaking Change (MAJOR)

````markdown
---
"chainlink-deployments-framework": major
---

feat!: redesign chain provider interface for multi-chain support

BREAKING CHANGE: The ChainProvider interface has been redesigned to support multiple blockchain networks simultaneously.

## Migration Guide

### Interface Changes (Optional)

**Before:**

```go
type ChainProvider interface {
    GetBalance(address string) (*big.Int, error)
    SendTransaction(tx *Transaction) error
}
```

**After:**

```go
type ChainProvider interface {
    GetBalance(ctx context.Context, chainID uint64, address string) (*big.Int, error)
    SendTransaction(ctx context.Context, chainID uint64, tx *Transaction) error
}
```

### Usage Migration

**Before:**

```go
provider := evm.NewProvider(config)
balance, err := provider.GetBalance("0x123...")
```

**After:**

```go
provider := evm.NewProvider(config)
ctx := context.Background()
balance, err := provider.GetBalance(ctx, 1, "0x123...")  // 1 = Ethereum mainnet
```
````

---

## Quick Reference

### Common Scenarios

| Change Type                | Version Bump | Example                                |
| -------------------------- | ------------ | -------------------------------------- |
| Add new public function    | MINOR        | `func NewFeature() Feature`            |
| Fix bug without API change | PATCH        | Internal logic correction              |
| Remove public function     | MAJOR        | Deleting exported function             |
| Change function signature  | MAJOR        | Adding required parameter              |
| Add optional struct field  | MINOR        | `Field *string` (pointer for optional) |
| Rename public struct field | MAJOR        | `OldName` → `NewName`                  |
| Update documentation       | PATCH        | README, code comments                  |
| Internal refactoring       | PATCH        | Private function changes               |
