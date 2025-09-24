---
"chainlink-deployments-framework": minor
---

Adds new convenience method `environment.New` to the test engine to bring up a new test environment

The `environment.New` method is a wrapper around the environment loading struct and allows the user
to load a new environment without having to instantiate the `Loader` struct themselves.

The `testing.T` argument has been removed and it's dependencies have been replaced with:

- A `context.Context` argument to the `Load` and `New` functions
- A new functional option `WithLogger` which overrides the default noop logger.

While this is a breaking change, the test environment is still in development and is not in actual usage yet.
