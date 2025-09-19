package database

import (
	"context"
	"fmt"
	"sort"

	"gorm.io/gorm"
)

type DatabaseMigration struct {
	gorm.Model
	Version int64 `gorm:"not null;uniqueIndex"`
}

type SchemaVersion struct {
	Migrations []int64 `json:"migrations"`
}

func NewSchemaVersion() SchemaVersion {
	sv := SchemaVersion{
		Migrations: []int64{
			1,
			0,
		},
	}
	sort.Slice(sv.Migrations, func(i, j int) bool {
		return sv.Migrations[i] < sv.Migrations[j]
	})
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
		if err := db.Exec("DROP SCHEMA IF EXISTS public CASCADE;").Error; err != nil {
			return fmt.Errorf("failed to drop public schema: %w", err)
		}
		if err := db.Exec("CREATE SCHEMA public;").Error; err != nil {
			return fmt.Errorf("failed to create public schema: %w", err)
		}
		if err := db.AutoMigrate(&DatabaseMigration{}); err != nil {
			return fmt.Errorf("failed to create 'database_migration' table: %w", err)
		}

		initialRecord := DatabaseMigration{Version: 0}
		if err := db.Create(&initialRecord).Error; err != nil {
			return fmt.Errorf("failed to insert initial migration record: %w", err)
		}
	}

	return nil
}

func (d *DBMigrator) lockVersion(ctx context.Context, tx *gorm.DB) (DatabaseMigration, error) {
	var m DatabaseMigration

	if err := tx.WithContext(ctx).
		Raw("SELECT id, version FROM migration_versions ORDER BY id LIMIT 1").
		Scan(&m).Error; err != nil {
		return m, err
	}

	if m.ID == 0 {
		return m, fmt.Errorf("no row found in migration_versions")
	}

	if err := tx.WithContext(ctx).
		Raw("SELECT id, version FROM migration_versions WHERE id = ? FOR UPDATE", m.ID).
		Scan(&m).Error; err != nil {
		return m, err
	}

	return m, nil
}

func (d *DBMigrator) Migrate() (err error) {
	if err = d.initialize(); err != nil {
		return err
	}
	migrations := NewSchemaVersion().Migrations
	ctx := context.Background()
	db := d.db
	tx := db.WithContext(ctx).Begin()
	// select for update
	currentVersion, err := d.lockVersion(ctx, tx)
	if err != nil {
		return
	}
	for _, migrationVersion := range migrations {
		if currentVersion.Version > migrationVersion {
			continue
		}

	}

	// release the select for update
	tx.Commit()
	return nil
}
