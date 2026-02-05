---
"chainlink-deployments-framework": patch
---

Rename all "Migration" terminology in domain layer methods to
"Changeset" for consistency with durable pipelines terminology.

EnvDir:

- MergeMigrationDataStore -> MergeChangesetDataStore
- MergeMigrationDataStoreCatalog -> MergeChangesetDataStoreCatalog
- MergeMigrationAddressBook -> MergeChangesetAddressBook
- RemoveMigrationAddressBook -> RemoveChangesetAddressBook

ArtifactsDir:

- MigrationDirPath -> ChangesetDirPath
- CreateMigrationDir -> CreateChangesetDir
- RemoveMigrationDir -> RemoveChangesetDir
- MigrationDirExists -> ChangesetDirExists
- MigrationOperationsReportsFileExists -> ChangesetOperationsReportsFileExists
- LoadAddressBookByMigrationKey -> LoadAddressBookByChangesetKey
- LoadDataStoreByMigrationKey -> LoadDataStoreByChangesetKey

Internal helpers:

- loadDataStoreByMigrationKey -> loadDataStoreByChangesetKey
- loadAddressBookByMigrationKey -> loadAddressBookByChangesetKey

Parameter renames: migKey/migkey -> csKey

BREAKING CHANGE: All public domain layer methods with "Migration" in
their name have been renamed to use "Changeset" instead. Update all
callers to use the new method names.
