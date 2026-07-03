package models

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

// migrationsFS 嵌入 migrations 目录下的所有 SQL 迁移文件，
// 使迁移逻辑随二进制分发，无需外部文件依赖。
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationsDir 是嵌入文件系统中迁移文件所在目录。
const migrationsDir = "migrations"

// RunMigrations 执行所有未应用的数据库迁移。
// 迁移文件以 pkg/models/migrations/*.sql 为唯一真源，由 goose 按文件名
// 版本号顺序执行，goose 自动维护 goose_db_version 表记录已应用版本。
//
// 调用前需确保数据库连接已建立且具备创建表/扩展的权限。
// 幂等：重复调用只会执行新增的迁移。
func RunMigrations(ctx context.Context, db *sql.DB) error {
	if dbLogger != nil {
		dbLogger.Debug("Running database migrations...")
	}

	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if dbLogger != nil {
		dbLogger.Debug("Database migrations applied successfully")
	}
	return nil
}

// MigrationStatus 返回当前迁移版本号与尚未应用的迁移列表，供诊断使用。
func MigrationStatus(ctx context.Context, db *sql.DB) (current int64, pending error) {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, fmt.Errorf("failed to set goose dialect: %w", err)
	}
	v, err := goose.GetDBVersionContext(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}
	return v, nil
}
