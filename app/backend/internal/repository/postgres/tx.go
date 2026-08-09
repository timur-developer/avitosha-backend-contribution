package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type txContextKey struct{}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if tx, ok := txFromContext(ctx); ok && tx != nil {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	committed := false

	defer func() {
		if committed {
			return
		}

		rollbackErr := tx.Rollback(context.Background())

		if recovered := recover(); recovered != nil {
			panic(recovered)
		}

		if rollbackErr == nil || errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return
		}

		if err != nil {
			err = errors.Join(err, fmt.Errorf("rollback tx: %w", rollbackErr))
			return
		}

		err = fmt.Errorf("rollback tx: %w", rollbackErr)
	}()

	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	committed = true
	return nil
}

func executorFromContext(ctx context.Context, fallback QueryExecutor) QueryExecutor {
	tx, ok := txFromContext(ctx)
	if ok && tx != nil {
		return tx
	}

	return fallback
}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}
