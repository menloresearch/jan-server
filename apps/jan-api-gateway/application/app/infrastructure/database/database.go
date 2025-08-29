package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

var SchemaRegistry []interface{}

func RegisterSchemaForAutoMigrate(models ...interface{}) {
	SchemaRegistry = append(SchemaRegistry, models...)
}

var DB *gorm.DB

func NewDB() (*gorm.DB, error) {
	if environment_variables.EnvironmentVariables.ENABLE_ADMIN_API {
		db, err := gorm.Open(postgres.Open(environment_variables.EnvironmentVariables.DB_POSTGRESQL_WRITE_DSN), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "5c16fb53-d98c-4fc6-8bb4-9abd3c0b9e88").
				Fatalf("unable to connect to database: %v", err)
			return nil, err
		}
		err = db.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{postgres.Open(
				environment_variables.EnvironmentVariables.DB_POSTGRESQL_READ1_DSN,
			)},
			Policy: dbresolver.RandomPolicy{},
		}))
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "9fab4b2e-1d70-4a4e-928a-5e81c7ee06de").
				Fatalf("unable to connect to setup replica: %v", err)
			return nil, err
		}

		// v0.0.6 only
		err = db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "5644d271-be84-4a92-a5af-489d87324758").
				Fatalf("unable to connect to setup replica: %v", err)
			return nil, err
		}
		err = db.Exec("CREATE SCHEMA public;").Error
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "07217a04-80f1-466f-8d2c-cdd162dd9ccb").
				Fatalf("unable to connect to setup replica: %v", err)
			return nil, err
		}
		for _, model := range SchemaRegistry {
			err = db.AutoMigrate(model)
			if err != nil {
				logger.GetLogger().
					WithField("error_code", "75333e43-8157-4f0a-8e34-aa34e6e7c285").
					Fatalf("failed to auto migrate schema: %T, error: %v", model, err)
				return nil, err
			}
		}

		DB = db
		return DB, nil
	}
	return nil, nil
}
