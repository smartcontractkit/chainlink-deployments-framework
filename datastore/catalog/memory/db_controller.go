package memory

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/proullon/ramsql/driver"
)

type DB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ DB = &dbController{}

func newDbController(db *sql.DB) *dbController {
	return &dbController{base: db}
}

type dbController struct {
	tx   *sql.Tx
	base *sql.DB
}

func (d *dbController) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if d.tx != nil {
		return d.tx.QueryContext(ctx, query, args...)
	}

	return d.base.QueryContext(ctx, query, args...)
}

func (d *dbController) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if d.tx != nil {
		return d.tx.ExecContext(ctx, query, args...)
	}

	return d.base.ExecContext(ctx, query, args...)
}

// Fixture performs an ExecContext but ignores the result, and is intended for test setup
func (d *dbController) Fixture(ctx context.Context, query string, args ...any) error {
	_, err := d.ExecContext(ctx, query, args...)
	return err
}

func (d *dbController) Begin(ctx context.Context) error {
	if d.tx != nil {
		return errors.New("transaction already started")
	}
	tx, err := d.base.BeginTx(ctx, nil)
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
