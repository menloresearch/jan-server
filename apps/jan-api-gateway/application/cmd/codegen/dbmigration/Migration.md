# Database Migration Procedure
---
We use [atlas](https://github.com/ariga/atlas) as our migrations tool.
Before you begin, please ensure your local environment is set up correctly:

To execute cmd/codegen/dbmigration, please check that:
1. Install Atlas: If you haven't already, install Atlas using Homebrew.
    ```
    brew install ariga/tap/atlas
    ```
2. Set up PostgreSQL: Ensure you have a local PostgreSQL instance running. Then, connect to it and set up the necessary user and database.
    ```sql
    CREATE ROLE migration WITH LOGIN PASSWORD 'migration';
    ALTER ROLE migration WITH SUPERUSER;
    CREATE DATABASE migration WITH OWNER = migration;
    ```
3. Configure Environment Variables: Set the following environment variables to point your application to the local database.
    ```
    export DB_POSTGRESQL_WRITE_DSN="host=localhost user=migration password=migration dbname=migration port=5432 sslmode=disable"
    export DB_POSTGRESQL_READ1_DSN="host=localhost user=migration password=migration dbname=migration port=5432 sslmode=disable"
    ```
---
The migration process is as follows (go run cmd/codegen/dbmigration):
1. Generate release.hcl: This file represents the current schema of your production database. It's your "from" schema.
```
func main() {
	environment_variables.EnvironmentVariables.LoadFromEnv()

	// git checkout main
	// generateHcl("main")

	// git checkout release
	generateHcl("release")

	// generateDiffSql()
}
```
2. Generate main.hcl: This file represents the desired new schema from your main branch. This is your "to" schema.
    ```
    func main() {
        environment_variables.EnvironmentVariables.LoadFromEnv()

        // git checkout main
        generateHcl("main")

        // git checkout release
        // generateHcl("release")

        // generateDiffSql()
    }
    ```
3. Create diff.sql: This command generates the SQL statements needed to migrate from the release schema to the main schema. The output is redirected to a file for review.
    ```
    func main() {
        environment_variables.EnvironmentVariables.LoadFromEnv()

        // git checkout main
        // generateHcl("main")

        // git checkout release
        // generateHcl("release")

        generateDiffSql()
    }
    ```
4. Validate diff.sql: This is a critical step. Open diff.sql and manually inspect the generated SQL for potentially harmful operations, such as:
    - Dropping columns: Look for DROP COLUMN statements. This is a destructive change and will result in permanent data loss.
    - Adding NOT NULL constraints: Directly adding a NOT NULL constraint to an existing column will fail if it contains NULL values. If Atlas generates this, you need to manually split the change into two safer steps (add nullable column, then update rows, and finally add the constraint).
