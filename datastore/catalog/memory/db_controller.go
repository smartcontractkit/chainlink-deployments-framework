package memory

import (
	"context"
	"database/sql"
	"errors"

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
	ctx := context.TODO()
	if d.tx != nil {
		return d.tx.QueryContext(ctx, q, args...)
	}

	return d.base.QueryContext(ctx, q, args...)
}

func (d *dbController) Exec(q string, args ...any) (sql.Result, error) {
	ctx := context.TODO()
	if d.tx != nil {
		return d.tx.ExecContext(ctx, q, args...)
	}

	return d.base.ExecContext(ctx, q, args...)
}

// Fixture performs an Exec but ignores the result, and is intended for test setup
func (d *dbController) Fixture(q string, args ...any) error {
	_, err := d.Exec(q, args...)
	return err
}

func (d *dbController) Begin() error {
	if d.tx != nil {
		return errors.New("transaction already started")
	}
	tx, err := d.base.BeginTx(context.TODO(), nil)
	if err != nil {
		return err
	}
	d.tx = tx

	return nil
}

func (d *dbController) Commit() error {
	if d.tx == nil {
		return errors.New("no transaction to commit")
	}
	defer func() {
		d.tx = nil
	}()

	return d.tx.Commit()
}

func (d *dbController) Rollback() error {
	if d.tx == nil {
		return errors.New("no transaction to roll back")
	}
	defer func() {
		d.tx = nil
	}()

	return d.tx.Rollback()
}
