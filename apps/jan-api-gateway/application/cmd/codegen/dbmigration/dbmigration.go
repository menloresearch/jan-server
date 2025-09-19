package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"menlo.ai/jan-api-gateway/app/infrastructure/database"
	_ "menlo.ai/jan-api-gateway/app/infrastructure/database/dbschema"
	"menlo.ai/jan-api-gateway/config/environment_variables"
)

// brew install ariga/tap/atlas
// postgres=# CREATE ROLE migration WITH LOGIN PASSWORD 'migration';
// postgres=# ALTER ROLE migration WITH SUPERUSER;
// postgres=# CREATE DATABASE migration WITH OWNER = migration;

func generateHcl(branchName string) {
	db, err := database.NewDB()
	if err != nil {
		panic(err)
	}
	err = db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error
	if err != nil {
		log.Fatalf("failed to drop schema: %v", err)
		return
	}
	err = db.Exec("CREATE SCHEMA public;").Error
	if err != nil {
		log.Fatalf("failed to create schema: %v", err)
		return
	}
	db.AutoMigrate(database.DatabaseMigration{})
	for _, model := range database.SchemaRegistry {
		err = db.AutoMigrate(model)
		if err != nil {
			panic(err)
		}
	}
	atlasCmdStr := `atlas schema inspect -u "postgres://migration:migration@localhost:5432/migration?sslmode=disable" > tmp/` + branchName + `.hcl`
	atlasCmd := exec.Command("sh", "-c", atlasCmdStr)
	atlasCmd.Run()
}

func generateDiffSql() {
	db, err := database.NewDB()
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

	atlasCmdStr := `atlas schema diff --dev-url "postgres://migration:migration@localhost:5432/migration?sslmode=disable" --from file://tmp/release.hcl --to file://tmp/main.hcl > tmp/diff.sql`
	atlasCmd := exec.Command("sh", "-c", atlasCmdStr)
	atlasCmd.Run()
}

func createTmpFolder() error {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
		return err
	}
	dirPath := fmt.Sprintf("%s/%s", dir, "tmp")
	err = os.MkdirAll(dirPath, 0755)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func main() {
	environment_variables.EnvironmentVariables.LoadFromEnv()
	if err := createTmpFolder(); err != nil {
		panic(err)
	}
	// git checkout main
	generateHcl("main")

	// git checkout release
	// generateHcl("release")

	// generateDiffSql()
}
