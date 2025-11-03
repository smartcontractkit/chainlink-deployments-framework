---
"chainlink-deployments-framework": minor
---

feat: add catalog service integration for datastore operations

Features:
- Add catalog service support for datastore management as alternative to local file storage
- Add `MergeMigrationDataStoreCatalog` method for catalog-based datastore persistence
- Existing `MergeMigrationDataStore` method continues to work for file-based storage (no breaking changes)
- Add unified `MergeDataStoreToCatalog` function for both initial migration and ongoing merge operations
- All catalog operations are transactional to prevent data inconsistencies
- Add `DatastoreType` configuration option (`file`/`catalog`) in domain.yaml to control storage backend
- Add new CLI command `datastore sync-to-catalog` for initial migration from file-based to catalog storage in CI
- Add `SyncDataStoreToCatalog` method to sync entire local datastore to catalog
- CLI automatically selects the appropriate merge method based on domain.yaml configuration
- Catalog mode does not modify local files - all updates go directly to the catalog service

Configuration:
- Set `datastore: catalog` in domain.yaml to enable catalog mode
- Set `datastore: file` or omit the setting to use traditional file-based storage
- CLI commands automatically detect the configuration and use the appropriate storage backend
