package repository

import "context"

// TxManager provides a simple transaction wrapper for repository operations.
// Implementations should start a DB transaction and pass a transaction-bound ctx to fn.
type TxManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
