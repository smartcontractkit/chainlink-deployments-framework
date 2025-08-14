# Remote implementation of Catalog datastore APIs

This implementation calls a gRPC service which is backed by the Catalog postgres
database. The service offers a streaming gRPC API, which allows for transaction
state to be maintained through that stream connection. If the stream closes,
normal cleanup will rollback the transaction. The [Catalog service APIs] include
message to begin, commit, and roll-back a transaction.

[Catalog service APIs]: http://github.com/smartcontractkit/chainlink-catalog
