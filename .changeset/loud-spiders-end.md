---
"chainlink-deployments-framework": major
---

feat!: add catalog service integration for datastore operations

BREAKING CHANGES:
- `EnvDir.MergeMigrationDataStore` now requires `context.Context` as first parameter and `datastore.CatalogStore` as last parameter
- Signature changed from `MergeMigrationDataStore(migkey, timestamp string)` to `MergeMigrationDataStore(ctx context.Context, migkey, timestamp string, catalog datastore.CatalogStore)`

Features:
- Add catalog service support for datastore management
- Add `SyncDataStoreToCatalog` function to push entire local datastore to catalog
- Add `MergeDataStoreToCatalog` function to merge migration datastores to catalog
- All catalog operations are transactional to prevent data inconsistencies
- Add `DatastoreType` configuration option to switch between `file` and `catalog` modes
