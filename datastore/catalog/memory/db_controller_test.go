package memory

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewControllerCommit(t *testing.T) {
	t.Parallel()
	db, stop := openMemDbForTest(t)
	defer stop()

	ctrl := newDbController(db)
	err := ctrl.Begin()
	require.NoError(t, err)
	_, err = ctrl.Exec("CREATE TABLE IF NOT EXISTS test(a int)")
	require.NoError(t, err)
	_, err = ctrl.Exec("INSERT INTO test (a) VALUES (1)")
	require.NoError(t, err)

	t.Run("Check inserted values", func(t *testing.T) {
		t.Parallel()
		rows, err2 := ctrl.Query("SELECT * FROM test")
		defer func(rows *sql.Rows) {
			assert.NoError(t, rows.Close())
		}(rows)
		require.NoError(t, err2)
		count := 0

		for rows.Next() {
			count++
		}
		require.NoError(t, err2)
		assert.Equal(t, 1, count)
	})

	t.Run("Check inserted values (outside of tx, so fail)", func(t *testing.T) {
		t.Parallel()
		_, err2 := ctrl.base.Query(`SELECT * FROM test`)
		require.ErrorContains(t, err2, `"test" does not exist`)
	})

	err = ctrl.Commit()
	require.NoError(t, err)
	assert.Nil(t, ctrl.tx)

	t.Run("Check inserted values (post-commit)", func(t *testing.T) {
		t.Parallel()
		rows, err2 := ctrl.Query("SELECT * FROM test")
		defer func(rows *sql.Rows) {
			assert.NoError(t, rows.Close())
		}(rows)
		require.NoError(t, err2)
		count := 0

		for rows.Next() {
			count++
		}
		require.NoError(t, err2)
		assert.Equal(t, 1, count)
	})
}

func TestNewControllerRollback(t *testing.T) {
	t.Parallel()
	db, stop := openMemDbForTest(t)
	defer stop()

	ctrl := newDbController(db)
	err := ctrl.Begin()
	require.NoError(t, err)
	_, err = ctrl.Exec("CREATE TABLE IF NOT EXISTS test(a int)")
	require.NoError(t, err)
	_, err = ctrl.Exec("INSERT INTO test (a) VALUES (1)")
	require.NoError(t, err)

	t.Run("Check inserted values", func(t *testing.T) {
		t.Parallel()
		rows, err2 := ctrl.Query("SELECT * FROM test")
		defer func(rows *sql.Rows) {
			assert.NoError(t, rows.Close())
		}(rows)
		require.NoError(t, err2)
		count := 0

		for rows.Next() {
			count++
		}
		require.NoError(t, err2)
		assert.Equal(t, 1, count)
	})

	t.Run("Check inserted values (outside of tx, so fail)", func(t *testing.T) {
		t.Parallel()
		_, err2 := ctrl.base.Query("SELECT * FROM test")
		require.ErrorContains(t, err2, `"test" does not exist`)
	})

	err = ctrl.Rollback()
	require.NoError(t, err)
	assert.Nil(t, ctrl.tx)

	t.Run("Check inserted values (post-rollback)", func(t *testing.T) {
		t.Parallel()
		_, err2 := ctrl.Query("SELECT * FROM test")
		require.ErrorContains(t, err2, `"test" does not exist`)
	})
}
