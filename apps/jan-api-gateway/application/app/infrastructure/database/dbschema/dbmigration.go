package dbschema

import (
	"menlo.ai/jan-api-gateway/app/infrastructure/database"
)

func init() {
	database.RegisterSchemaForAutoMigrate(DatabaseMigration{})
}

type DatabaseMigration struct {
	BaseModel
	Version int64 `gorm:"not null;uniqueIndex"`
}
