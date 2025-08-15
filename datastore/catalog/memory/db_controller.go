package memory

import (
	"database/sql"
	"fmt"

	_ "github.com/proullon/ramsql/driver"
)

type DB interface {
	Query(q string, args ...any) (*sql.Rows, error)
	Exec(q string, args ...any) (sql.Result, error)
}

var _ DB = &dbController{}

func newDbController(db *sql.DB) *dbController {
	return &dbController{base: db}
}

type dbController struct {
	tx   *sql.Tx
	base *sql.DB
}

func (d *dbController) Query(q string, args ...any) (*sql.Rows, error) {
	fmt.Println("Executing query ", append([]any{q}, args))
	if d.tx != nil {
		return d.tx.Query(q, args...)
	}
	return d.base.Query(q, args...)
}

func (d *dbController) Exec(q string, args ...any) (sql.Result, error) {
	fmt.Println("Executing statement ", append([]any{q}, args))
	if d.tx != nil {
		return d.tx.Exec(q, args...)
	}
	return d.base.Exec(q, args...)
}

// Fixture performs an Exec but ignores the result, and is intended for test setup
func (d *dbController) Fixture(q string, args ...any) error {
	_, err := d.Exec(q, args...)
	return err
}

func (d *dbController) Begin() error {
	if d.tx != nil {
		return fmt.Errorf("transaction already started")
	}
	tx, err := d.base.Begin()
	if err != nil {
		return err
	}
	d.tx = tx
	return nil
}

func (d *dbController) Commit() error {
	if d.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}
	defer func() {
		d.tx = nil
	}()
	return d.tx.Commit()
}

func (d *dbController) Rollback() error {
	if d.tx == nil {
		return fmt.Errorf("no transaction to roll back")
	}
	defer func() {
		d.tx = nil
	}()
	return d.tx.Rollback()
}
