<div align="center">
  <h1>Chainlink Deployments Framework</h1>
  <a><img src="https://github.com/smartcontractkit/chainlink-deployments-framework/actions/workflows/push-main.yml/badge.svg" /></a>
  <br/>
  <br/>
</div>


This repository contains the Chainlink Deployments Framework, a comprehensive set of libraries that enables developers to build, manage, and execute(in future) deployment changesets. 
The framework includes the Operations API and Datastore API.

## Usage

```bash
$ go get github.com/smartcontractkit/chainlink-deployments-framework
```

## Imports

```go
import (
  "github.com/smartcontractkit/chainlink-deployments-framework/deployment" // for writing changesets (migrated from chainlink/deployments
  "github.com/smartcontractkit/chainlink-deployments-framework/operations" // for operations API
  "github.com/smartcontractkit/chainlink-deployments-framework/datastore" // for datastore API
)
```

## Development


### Installing Dependencies

Install the required tools using [asdf](https://asdf-vm.com/guide/getting-started.html):

```bash
asdf install
```

### Linting

```bash
task lint
```

### Testing

```bash
task test
```

## Contributing

For instructions on how to contribute to `chainlink-deployments-framework` and the release process,
see [CONTRIBUTING.md](https://github.com/smartcontractkit/chainlink-deployments-framework/blob/main/CONTRIBUTING.md)

## Releasing

For instructions on how to release `chainlink-deployments-framework`,
see [RELEASE.md](https://github.com/smartcontractkit/chainlink-deployments-framework/blob/main/RELEASE.md)
