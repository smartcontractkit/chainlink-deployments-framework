# operations-gen

Generates type-safe Go operation wrappers for smart contracts from their ABIs.

## Usage

From inside this repository:

```bash
go run ./tools/operations-gen -config /path/to/operations_gen_config.yaml
```

From any other repository, install the binary once and invoke it directly:

```bash
# Pin to a published subdirectory tag
go install github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen@vX.Y.Z

operations-gen -config /path/to/operations_gen_config.yaml
```

The `-config` path can be absolute or relative to the current working directory. All paths inside the config (ABI dirs, bytecode dirs, output dir) are resolved relative to the config file's directory, so the binary can be run from anywhere.

Print the CLI release metadata:

```bash
operations-gen -version
```

## Install a released version

This module is released with Go module subdirectory tags in the form `tools/operations-gen/vX.Y.Z`.

Install a specific released version:

```bash
go install github.com/smartcontractkit/chainlink-deployments-framework/tools/operations-gen@vX.Y.Z
```

Download prebuilt release binaries and checksums from the GitHub Releases page:

```text
https://github.com/smartcontractkit/chainlink-deployments-framework/releases
```

## Project structure

```text
tools/operations-gen/
  main.go                         # CLI entrypoint + chain-family dispatch
  templates/
    evm/
      operations.tmpl             # EVM codegen template
  internal/
    core/
      core.go                     # Shared config + helpers/interfaces
    families/
      evm/
        evm.go                    # EVM handler implementation
        abi.go, contract.go,
        codegen.go                # ABI → IR → template-data pipeline
        *_test.go                 # EVM unit tests
        evm_golden_test.go        # End-to-end golden generation tests
  testdata/
    evm/                          # ABI/bytecode/config/golden fixtures
```

`main.go` intentionally stays thin: it parses top-level config, loads the template for the selected chain family, and dispatches to the family handler. Shared helpers and common config types live in `internal/core`.

## Configuration

Create an `operations_gen_config.yaml` that points at your abigen-generated gobindings package:

```yaml
version: "1.0.0"
chain_family: evm # Optional: defaults to "evm"

output:
  base_path: "." # Directory where generated operations/ folders are written

input:
  gobindings_package: "github.com/smartcontractkit/chainlink-ccip/chains/evm/gobindings/generated"

contracts:
  - contract_name: FeeQuoter
    version: "1.6.0"
    package_name: fee_quoter # Optional: override default package name
    omit_deploy: false # Optional: set true to skip Deploy operation generation (default: false)
    functions:
      - name: updatePrices
        access: owner # Write op with MCMS support
      - name: getTokenPrice
        access: public # Read op (or public write op)
```

### Top-level fields

| Field                      | Required | Description                                                                                                                                                 |
| -------------------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `version`                  | Yes      | Config schema version                                                                                                                                       |
| `chain_family`             | No       | Target chain family. Only `"evm"` is supported. Defaults to `"evm"`.                                                                                        |
| `input.gobindings_package` | No       | Parent Go import path containing versioned abigen packages. Used to derive contract bindings as `<input.gobindings_package>/<version_path>/<package_name>`. |
| `output.base_path`         | Yes      | Root directory where generated files are written. Relative to the config file.                                                                              |

### Contract fields

| Field                | Required | Description                                                                                                                                            |
| -------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `contract_name`      | Yes      | Contract name as it appears in the ABI (e.g. `FeeQuoter`)                                                                                              |
| `version`            | Yes      | Semver version of the contract (e.g. `"1.6.0"`)                                                                                                        |
| `gobindings_package` | No       | Optional full Go import path override for this contract's abigen-generated bindings package. Required only when `input.gobindings_package` is not set. |
| `package_name`       | No       | Override the generated Go package name. Defaults to `snake_case(contract_name)`.                                                                       |
| `version_path`       | No       | Override the directory path derived from the version. Defaults to `v{major}_{minor}_{patch}`.                                                          |
| `omit_deploy`        | No       | Skip generation of the `Deploy` operation and bytecode constant. Defaults to `false`.                                                                  |

### Function access control

| Value    | Behaviour                                                                                                                          |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `owner`  | Generates a write operation gated by `OnlyOwner`, producing an MCMS-compatible transaction when the deployer key is not the owner. |
| `public` | Generates a read operation (for `view`/`pure` functions) or an unrestricted write operation.                                       |

## Gobindings requirements

The generator expects an abigen-generated package that exports the standard metadata symbol:

```go
var FeeQuoterMetaData = &bind.MetaData{
    ABI: "...",
    Bin: "...",
}
```

For `omit_deploy: true`, only the `ABI` field is required. Otherwise both `ABI` and `Bin` must be present.

## Output layout

Generated files are written to:

```
{output.base_path}/
  v1_6_0/
    operations/
      fee_quoter/
        fee_quoter.go
```

Each generated file contains:

- ABI and bytecode constants, plus a `ContractType` and `Version`
- A `Deploy` operation (unless `omit_deploy: true`)
- A `NewWrite<Fn>(c gobindings.<Contract>Interface)` factory for every `access: owner` (or writable `access: public`) function, returning `*cld_ops.Operation[…]`
- A `NewRead<Fn>(c gobindings.<Contract>Interface)` factory for every `view` / `pure` function
- `*Args` structs for functions that take multiple inputs, and `*Result` structs for reads that return multiple outputs

The generator does not emit its own contract wrapper: each factory takes an interface from the abigen-generated `gobindings` package. The caller is expected to bind the contract via that package (`gobindings.New<Contract>(addr, backend)`) and hand the result in.

The generated code depends on three imports:

- `github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/operations/contract` — the operations runtime
- `github.com/smartcontractkit/chainlink-deployments-framework/chain/evm` and `.../operations` — chain + ops types used in the factory signatures
- `{gobindings_package}` — the derived or per-contract override abigen bindings import path

## Extending to new chain families

> Only `evm` is supported today. The steps below describe how to add support for a new family in the future.

The generator dispatches entirely by `chain_family`. Each family owns its own YAML contract schema, type mappings, template, and generation logic; only common CLI/config plumbing and dispatch utilities are shared.

To add a new chain family (e.g. `solana`):

1. Create `internal/families/solana/solana.go` with a `solana.Handler` type implementing `core.ChainFamilyHandler`:

   ```go
   type ChainFamilyHandler interface {
       Generate(config core.Config, tmpl *template.Template) error
   }
   ```

   The handler receives the full `core.Config`. `Config.Input`, `Config.Output`, and `Config.Contracts` are `yaml.Node` values so each chain-family handler can decode its own chain-specific schemas.

2. Add `templates/solana/operations.tmpl` with chain-appropriate imports and method bodies.

3. Register the handler in `chainFamilies` in `main.go`:
   ```go
   var chainFamilies = map[string]core.ChainFamilyHandler{
       "evm":    evm.Handler{},
       "solana": solana.Handler{},
   }
   ```

No other changes to `main.go` are needed. Set `chain_family: solana` in your config to use it.
