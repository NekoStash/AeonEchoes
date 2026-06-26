package postgres

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

type migrationFile struct {
	Version  string
	Name     string
	SQL      string
	Checksum string
}

// RunMigrations applies SQL migrations once and records checksums in schema_migrations.
// If migrationsDir is empty, embedded migrations are used.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	migrations, err := loadMigrations(migrationsDir)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return fmt.Errorf("no postgres migrations found")
	}
	if _, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	for _, migration := range migrations {
		if err := applyMigration(ctx, pool, migration); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, migration migrationFile) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Name, err)
	}
	defer tx.Rollback(ctx)

	var existingChecksum string
	err = tx.QueryRow(ctx, `SELECT checksum FROM schema_migrations WHERE version = $1`, migration.Version).Scan(&existingChecksum)
	if err == nil {
		if existingChecksum != migration.Checksum {
			return fmt.Errorf("migration %s checksum mismatch: database has %s, file has %s", migration.Name, existingChecksum, migration.Checksum)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit already-applied migration %s: %w", migration.Name, err)
		}
		return nil
	}
	if !isNoRows(err) {
		return fmt.Errorf("read schema_migrations for %s: %w", migration.Name, err)
	}
	if _, err := tx.Exec(ctx, migration.SQL); err != nil {
		return fmt.Errorf("execute migration %s: %w", migration.Name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version, name, checksum) VALUES ($1, $2, $3)`, migration.Version, migration.Name, migration.Checksum); err != nil {
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Name, err)
	}
	return nil
}

func loadMigrations(migrationsDir string) ([]migrationFile, error) {
	if strings.TrimSpace(migrationsDir) == "" {
		entries, err := embeddedMigrations.ReadDir("migrations")
		if err != nil {
			return nil, fmt.Errorf("read embedded migrations: %w", err)
		}
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			files = append(files, entry.Name())
		}
		sort.Strings(files)
		migrations := make([]migrationFile, 0, len(files))
		for _, name := range files {
			data, err := embeddedMigrations.ReadFile(filepath.ToSlash(filepath.Join("migrations", name)))
			if err != nil {
				return nil, fmt.Errorf("read embedded migration %s: %w", name, err)
			}
			migration, err := parseMigration(name, data)
			if err != nil {
				return nil, err
			}
			migrations = append(migrations, migration)
		}
		return migrations, nil
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)
	migrations := make([]migrationFile, 0, len(files))
	for _, name := range files {
		path := filepath.Join(migrationsDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", path, err)
		}
		migration, err := parseMigration(name, data)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration)
	}
	return migrations, nil
}

func parseMigration(name string, data []byte) (migrationFile, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return migrationFile{}, fmt.Errorf("migration file %q must start with numeric version and underscore", name)
	}
	sql := strings.TrimSpace(string(data))
	if sql == "" {
		return migrationFile{}, fmt.Errorf("migration file %q is empty", name)
	}
	sum := sha256.Sum256(data)
	return migrationFile{Version: parts[0], Name: name, SQL: sql, Checksum: hex.EncodeToString(sum[:])}, nil
}
