package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"menlo.ai/jan-api-gateway/app/utils/logger"
)

type DatabaseMigration struct {
	gorm.Model
	Version string `gorm:"not null;uniqueIndex"`
}

type SchemaVersion struct {
	Migrations []string `json:"migrations"`
}

func NewSchemaVersion() SchemaVersion {
	sv := SchemaVersion{
		// Consider supporting semantic versioning, such as:
		// ```
		// Version {
		//   ReleaseVersion: "v0.0.3",
		//   DbVersion: 2
		// }
		// ```
		Migrations: []string{
			"000001",
			"000002",
		},
	}
	return sv
}

type DBMigrator struct {
	db *gorm.DB
}

func NewDBMigrator(db *gorm.DB) *DBMigrator {
	return &DBMigrator{
		db: db,
	}
}

func (d *DBMigrator) initialize() error {
	db := d.db
	var reset bool
	var record DatabaseMigration

	hasTable := db.Migrator().HasTable("database_migration")
	if hasTable {
		result := db.Limit(1).Find(&record)
		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to query migration records: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			reset = true
		}
	} else {
		reset = true
	}

	if reset {
		// Still experiencing a race condition here, need to consult with DevOps regarding deployment strategy.
		if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error; err != nil {
			return fmt.Errorf("failed to drop public schema: %w", err)
		}
		if err := db.Exec("CREATE SCHEMA public;").Error; err != nil {
			return fmt.Errorf("failed to create public schema: %w", err)
		}
		if err := db.AutoMigrate(&DatabaseMigration{}); err != nil {
			return fmt.Errorf("failed to create 'database_migration' table: %w", err)
		}

		initialRecord := DatabaseMigration{Version: "000000"}
		if err := db.Create(&initialRecord).Error; err != nil {
			return fmt.Errorf("failed to insert initial migration record: %w", err)
		}
	}

	return nil
}

func (d *DBMigrator) lockVersion(ctx context.Context, tx *gorm.DB) (DatabaseMigration, error) {
	var m DatabaseMigration

	if err := tx.WithContext(ctx).
		Raw("SELECT id, version FROM database_migration ORDER BY id LIMIT 1").
		Scan(&m).Error; err != nil {
		return m, err
	}

	if m.ID == 0 {
		return m, fmt.Errorf("no row found in database_migration")
	}

	if err := tx.WithContext(ctx).
		Raw("SELECT id, version FROM database_migration WHERE id = ? FOR UPDATE", m.ID).
		Scan(&m).Error; err != nil {
		return m, err
	}

	return m, nil
}

func (d *DBMigrator) Migrate() (err error) {
	if err = d.initialize(); err != nil {
		return err
	}
	for _, model := range SchemaRegistry {
		err = d.db.AutoMigrate(model)
		if err != nil {
			logger.GetLogger().
				WithField("error_code", "75333e43-8157-4f0a-8e34-aa34e6e7c285").
				Fatalf("failed to auto migrate schema: %T, error: %v", model, err)
			return err
		}
	}
	return nil
}

// func (d *DBMigrator) Migrate() (err error) {
// 	if err = d.initialize(); err != nil {
// 		return err
// 	}
// 	migrations := NewSchemaVersion().Migrations
// 	ctx := context.Background()
// 	db := d.db
// 	tx := db.WithContext(ctx).Begin()
// 	// select for update
// 	currentVersion, err := d.lockVersion(ctx, tx)
// 	if err != nil {
// 		return
// 	}
// 	_, filename, _, ok := runtime.Caller(0)
// 	if !ok {
// 		return fmt.Errorf("da75e6a4-af0e-46a0-8cf8-569263651443")
// 	}
// 	migrationSqlFolder := filepath.Join(filepath.Dir(filename), "migrationsqls")

// 	updated := false
// 	for _, migrationVersion := range migrations {
// 		if currentVersion.Version >= migrationVersion {
// 			continue
// 		}
// 		// get version sql file
// 		sqlFile := filepath.Join(migrationSqlFolder, fmt.Sprintf("%s.sql", migrationVersion))
// 		content, err := os.ReadFile(sqlFile)
// 		if err != nil {
// 			return err
// 		}

// 		fileContentAsString := string(content)
// 		sqlCommands := strings.Split(fileContentAsString, ";")
// 		for _, command := range sqlCommands {
// 			db.Exec(command)
// 		}
// 		updated = true
// 	}
// 	if updated {
// 		currentVersion.Version = migrations[len(migrations)-1]
// 		if err := tx.Save(currentVersion).Error; err != nil {
// 			tx.Rollback()
// 			return err
// 		}
// 	}
// 	tx.Commit()
// 	return nil
// }
