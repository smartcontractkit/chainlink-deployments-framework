package domain

import (
	"errors"
	"fmt"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// LoadDataStoreByMigrationKey searches for a datastore file in the migration directory and
// returns the datastore as read-only.
//
// The search will look for a datastore file with a matching name as the domain, env and
// migration key, returning the first matching file. An error is returned if no matches are found
// or if an error occurs during the search.
//
// Pattern format: "*-<domain>-<env>-<migKey>_datastore.json".
func (a *ArtifactsDir) LoadDataStoreByMigrationKey(migKey string) (fdatastore.DataStore, error) {
	migDirPath := a.MigrationDirPath(migKey)
	pattern := fmt.Sprintf("*-%s-%s-%s_%s",
		a.DomainKey(), a.EnvKey(), migKey, DataStoreFileName,
	)

	dataStorePath, err := a.findArtifactPath(migDirPath, pattern)
	if err != nil {
		return nil, err
	}

	return a.loadDataStore(dataStorePath)
}

func loadDataStoreByMigrationKey(artDir *ArtifactsDir, migKey, timestamp string) (fdatastore.DataStore, error) {
	// Set the durable pipelines directory and timestamp if provided
	if timestamp != "" {
		if err := artDir.SetDurablePipelines(timestamp); err != nil {
			return nil, err
		}
	}

	// Load the migration datastore where the artifacts group name is the migration key
	migDataStore, err := artDir.LoadDataStoreByMigrationKey(migKey)
	if err != nil {
		if errors.Is(err, ErrArtifactNotFound) {
			fmt.Println("No migration data store found, skipping merge")

			return fdatastore.NewMemoryDataStore().Seal(), nil
		}

		return fdatastore.NewMemoryDataStore().Seal(), err
	}

	return migDataStore, nil
}
