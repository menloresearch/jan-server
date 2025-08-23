package main

import (
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"

	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	_ "menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

var GormGenerator *gen.Generator

func init() {
	environment_variables.EnvironmentVariables.LoadFromEnv()
	db, err := gorm.Open(postgres.Open(environment_variables.EnvironmentVariables.DB_POSTGRESQL_WRITE_DSN))
	if err != nil {
		panic(err)
	}

	GormGenerator = gen.NewGenerator(gen.Config{
		OutPath:       "./app/infrastructure/database/gormgen",
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		FieldNullable: true,
	})
	GormGenerator.UseDB(db)
}

func main() {
	for _, model := range database.SchemaRegistry {
		GormGenerator.ApplyBasic(model)
		type Querier interface {
		}
		GormGenerator.ApplyInterface(func(Querier) {}, model)
	}
	GormGenerator.Execute()
}
