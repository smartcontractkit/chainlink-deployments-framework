package memory

import (
	"database/sql"
	"testing"

	"github.com/rubenv/pgtest"
	"github.com/stretchr/testify/require"
)

// openMemDbForTest opens an in-memory postgres
func openMemDbForTest(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	pg, err := pgtest.Start()
	require.NoError(t, err)

	return pg.DB, func() {
		require.NoError(t, pg.Stop())
	}
}
