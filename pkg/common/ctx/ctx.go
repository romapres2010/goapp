package ctx

import (
	"context"

	"github.com/jmoiron/sqlx" // https://jmoiron.github.io/sqlx/
)

const DUMMY_VAL_FROM_CTX = 999999999

// The key type for Context value
type key int

const httpRequestIDKey key = 0
const txKey key = 1
const sqlKey key = 2

// NewContextHTTPRequestID returns a new Context carrying RequestID.
func NewContextHTTPRequestID(ctx context.Context, requestID uint64) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, httpRequestIDKey, requestID)
	}
	return nil
}

// FromContextHTTPRequestID extracts the RequestID from ctx, if present.
func FromContextHTTPRequestID(ctx context.Context) uint64 {
	if ctx != nil {
		val, ok := ctx.Value(httpRequestIDKey).(uint64)
		if !ok {
			return 0
		}
		return val
	}
	return uint64(DUMMY_VAL_FROM_CTX)
}

// FromContextTx extracts the Tx from ctx, if present.
func FromContextTx(ctx context.Context) *sqlx.Tx {
	if ctx != nil {
		tx, ok := ctx.Value(txKey).(*sqlx.Tx)
		if !ok {
			return nil
		}
		return tx
	}
	return nil
}

// NewContextTx returns a new Context carrying Tx.
func NewContextTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	if ctx != nil {
		return context.WithValue(ctx, txKey, tx)
	}
	return nil
}

// FromContextSQLId extracts the SQL id from ctx, if present
func FromContextSQLId(ctx context.Context) uint64 {
	if ctx != nil {
		val, ok := ctx.Value(sqlKey).(uint64)
		if !ok {
			return 0
		}
		return val
	}
	return uint64(DUMMY_VAL_FROM_CTX)
}

// NewContextSQLId returns a new Context carrying SQL id
func NewContextSQLId(ctx context.Context, sqlID uint64) context.Context {
	return context.WithValue(ctx, sqlKey, sqlID)
}
