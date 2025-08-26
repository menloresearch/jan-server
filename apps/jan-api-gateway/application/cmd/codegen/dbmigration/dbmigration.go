package main

import (
	"log"
	"os/exec"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	_ "menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

// brew install ariga/tap/atlas
// postgres=# CREATE ROLE migration WITH LOGIN PASSWORD 'migration';
// postgres=# ALTER ROLE migration WITH SUPERUSER;
// postgres=# CREATE DATABASE migration WITH OWNER = migration;

func generateHcl(branchName string) {
	db, err := gorm.Open(postgres.Open("host=localhost user=migration dbname=migration port=5432 sslmode=disable"))
	if err != nil {
		panic(err)
	}
	err = db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error
	if err != nil {
		log.Fatalf("failed to drop schema: %v", err)
	}
	err = db.Exec("CREATE SCHEMA public;").Error
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	for _, model := range database.SchemaRegistry {
		err := db.AutoMigrate(model)
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "75333e43-8157-4f0a-8e34-aa34e6e7c285").
				Fatalf("failed to auto migrate schema: %T, error: %v", model, err)
		}
	}
	atlasCmdStr := `atlas schema inspect -u "postgres://migration:migration@localhost:5432/migration?sslmode=disable" > tmp/` + branchName + `.hcl`
	atlasCmd := exec.Command("sh", "-c", atlasCmdStr)
	atlasCmd.Run()
}

func generateDiffSql() {
	db, err := gorm.Open(postgres.Open("host=localhost user=migration dbname=migration port=5432 sslmode=disable"))
	if err != nil {
		panic(err)
	}
	err = db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error
	if err != nil {
		log.Fatalf("failed to drop schema: %v", err)
	}
	err = db.Exec("CREATE SCHEMA public;").Error
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}

	atlasCmdStr := `atlas schema diff --dev-url "postgres://migration:migration@localhost:5432/migration?sslmode=disable" --from file://tmp/main.hcl --to file://tmp/release.hcl > tmp/diff.sql`
	atlasCmd := exec.Command("sh", "-c", atlasCmdStr)
	atlasCmd.Run()
}

func main() {
	environment_variables.EnvironmentVariables.LoadFromEnv()

	// git checkout main
	// generateHcl("main")

	// git checkout release
	// generateHcl("release")

	// generateDiffSql()
}
