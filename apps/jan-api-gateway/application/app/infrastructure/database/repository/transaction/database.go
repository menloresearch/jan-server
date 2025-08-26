package transaction

import (
	"context"

	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/app/infrastructure/database/gormgen"
	"menlo.ai/jan-api-gateway/app/utils/contextkeys"
)

func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, contextkeys.TransactionContextKey{}, tx)
}

type Database struct {
	db *gorm.DB
}

func (t *Database) GetTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(contextkeys.TransactionContextKey{}).(*gorm.DB); ok {
		return tx
	}
	return t.db
}

func (t *Database) GetQuery(ctx context.Context) *gormgen.Query {
	db := t.GetTx(ctx)
	return gormgen.Use(db)
}

func NewDatabase(db *gorm.DB) *Database {
	return &Database{db}
}
