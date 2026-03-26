package domain

import (
	"errors"
	"fmt"

	fdatastore "github.com/smartcontractkit/chainlink-deployments-framework/datastore"
)

// LoadDataStoreByChangesetKey searches for a datastore file in the changeset directory and
// returns the datastore as read-only.
//
// The search will look for a datastore file with a matching name as the domain, env and
// changeset key, returning the first matching file. An error is returned if no matches are found
// or if an error occurs during the search.
//
// Pattern format: "*-<domain>-<env>-<csKey>_datastore.json".
func (a *ArtifactsDir) LoadDataStoreByChangesetKey(csKey string) (fdatastore.DataStore, error) {
	csDirPath := a.ChangesetDirPath(csKey)
	pattern := fmt.Sprintf("*-%s-%s-%s_%s",
		a.DomainKey(), a.EnvKey(), csKey, DataStoreFileName,
	)

	dataStorePath, err := a.findArtifactPath(csDirPath, pattern)
	if err != nil {
		return nil, err
	}

	return a.loadDataStore(dataStorePath)
}

func loadDataStoreByChangesetKey(artDir *ArtifactsDir, csKey, timestamp string) (fdatastore.DataStore, error) {
	// Set the durable pipelines directory and timestamp if provided
	if timestamp != "" {
		if err := artDir.SetDurablePipelines(timestamp); err != nil {
			return nil, err
		}
	}

	// Load the changeset datastore where the artifacts group name is the changeset key
	csDataStore, err := artDir.LoadDataStoreByChangesetKey(csKey)
	if err != nil {
		if errors.Is(err, ErrArtifactNotFound) {
			fmt.Println("No changeset data store found, skipping merge")

			return fdatastore.NewMemoryDataStore().Seal(), nil
		}

		return fdatastore.NewMemoryDataStore().Seal(), err
	}

	return csDataStore, nil
}
