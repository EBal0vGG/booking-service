package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

func (m *TxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	txCtx := withTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		if rbErr := tx.Rollback(txCtx); rbErr != nil {
			return fmt.Errorf("tx rollback failed: %v (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
