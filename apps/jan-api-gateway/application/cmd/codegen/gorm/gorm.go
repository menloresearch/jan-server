package main

import (
	"gorm.io/driver/postgres"
	"gorm.io/gen"
	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/cmd/codegen/gorm/models/user"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func main() {
	environment_variables.EnvironmentVariables.LoadFromEnv()
	db, err := gorm.Open(postgres.Open(environment_variables.EnvironmentVariables.DB_POSTGRESQL_WRITE_DSN))
	if err != nil {
		panic(err)
	}

	g := gen.NewGenerator(gen.Config{
		OutPath:       "./app/infrastructure/database/gormgen",
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface | gen.WithoutContext,
		FieldNullable: true,
	})

	g.UseDB(db)
	user.RegisterUser(g)
	g.Execute()
}
