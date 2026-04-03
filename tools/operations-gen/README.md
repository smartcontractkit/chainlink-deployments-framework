# operations-gen

Generates type-safe Go operation wrappers for smart contracts from their ABIs.

## Usage

Via the repo task runner (from the repo root):

```bash
task generate:operations CONFIG=/path/to/operations_gen_config.yaml
```

Or run directly from this directory:

```bash
go run . -config /path/to/operations_gen_config.yaml
```

The `-config` path can be absolute or relative to the current working directory.

## Configuration

Create an `operations_gen_config.yaml` alongside your ABI/bytecode directories:

```yaml
version: "1.0.0"
chain_family: evm # Optional: defaults to "evm"

input:
  base_path: "." # Directory containing abi/ and bytecode/ subdirectories

output:
  base_path: "." # Directory where generated operations/ folders are written

contracts:
  - contract_name: FeeQuoter
    version: "1.6.0"
    package_name: fee_quoter # Optional: override default package name
    abi_file: "fee_quoter.json" # Optional: override default ABI filename
    no_deployment: false # Optional: skip bytecode and Deploy operation
    functions:
      - name: updatePrices
        access: owner # Write op with MCMS support
      - name: getTokenPrice
        access: public # Read op (or public write op)
```

### Top-level fields

| Field              | Required | Description                                                                                    |
| ------------------ | -------- | ---------------------------------------------------------------------------------------------- |
| `version`          | Yes      | Config schema version                                                                          |
| `chain_family`     | No       | Target chain family. Only `"evm"` is supported. Defaults to `"evm"`.                          |
| `input.base_path`  | Yes      | Root directory containing `abi/` and `bytecode/` subdirectories. Relative to the config file. |
| `output.base_path` | Yes      | Root directory where generated files are written. Relative to the config file.                 |

### Contract fields

| Field           | Required | Description                                                                                                      |
| --------------- | -------- | ---------------------------------------------------------------------------------------------------------------- |
| `contract_name` | Yes      | Contract name as it appears in the ABI (e.g. `FeeQuoter`)                                                        |
| `version`       | Yes      | Semver version of the contract (e.g. `"1.6.0"`)                                                                  |
| `package_name`  | No       | Override the generated Go package name. Defaults to `snake_case(contract_name)`.                                 |
| `abi_file`      | No       | Override the ABI filename. Defaults to `{package_name}.json`.                                                    |
| `version_path`  | No       | Override the directory path derived from the version. Defaults to `v{major}_{minor}_{patch}`.                    |
| `no_deployment` | No       | Skip the bytecode constant and `Deploy` operation. Useful for contracts deployed elsewhere. Defaults to `false`. |

### Function access control

| Value    | Behaviour                                                                                                                          |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| `owner`  | Generates a write operation gated by `OnlyOwner`, producing an MCMS-compatible transaction when the deployer key is not the owner. |
| `public` | Generates a read operation (for `view`/`pure` functions) or an unrestricted write operation.                                       |

## Input layout

The generator expects ABIs and bytecode at fixed paths under `input.base_path`:

```
{input.base_path}/
  abi/
    v1_6_0/
      fee_quoter.json
  bytecode/
    v1_6_0/
      fee_quoter.bin
```

Version `1.6.0` maps to directory `v1_6_0`. Override with `version_path` if your layout differs.

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

- ABI and bytecode constants
- A bound contract wrapper with typed methods
- A `Deploy` operation (unless `no_deployment: true`)
- A typed write operation for each `access: owner` or writable `access: public` function
- A typed read operation for each `view`/`pure` function

The generated code imports the runtime helpers from:

```
github.com/smartcontractkit/chainlink-deployments-framework/chain/evm/operations/contract
```

## Extending to new chain families

> Only `evm` is supported today. The steps below describe how to add support for a new family in the future.

The generator dispatches entirely by `chain_family`. Each family owns its own YAML contract schema, type mappings, template, and generation logic — nothing is shared between families at the code level.

To add a new chain family (e.g. `solana`):

1. Create `solana.go` with a `solanaHandler` struct implementing `ChainFamilyHandler`:
   ```go
   type ChainFamilyHandler interface {
       Generate(config Config, tmpl *template.Template) error
   }
   ```
   The handler receives the full `Config` (including raw `[]*yaml.Node` contracts) and decodes its own chain-specific contract schema from those nodes.

2. Add `templates/solana/operations.tmpl` with chain-appropriate imports and method bodies.

3. Register the handler in `chainFamilies` in `main.go`:
   ```go
   var chainFamilies = map[string]ChainFamilyHandler{
       "evm":    evmHandler{},
       "solana": solanaHandler{},
   }
   ```

No other changes to `main.go` are needed. Set `chain_family: solana` in your config to use it.
