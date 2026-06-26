package postgres

import (
	"context"
	"fmt"
	"strings"

	"aeonechoes/server/internal/domain"
)

func (s *Store) GetSetting(scope, key string) (domain.AppSetting, error) {
	if err := requireStore(s); err != nil {
		return domain.AppSetting{}, err
	}
	cleanScope := strings.TrimSpace(scope)
	cleanKey := strings.TrimSpace(key)
	if cleanScope == "" || cleanKey == "" {
		return domain.AppSetting{}, fmt.Errorf("setting scope and key must not be empty")
	}
	row := s.pool.QueryRow(context.Background(), `SELECT key, value, updated_at FROM settings WHERE key=$1`, settingStorageKey(cleanScope, cleanKey))
	item, err := scanSetting(row)
	if err != nil {
		if isNoRows(err) {
			return domain.AppSetting{}, fmt.Errorf("setting %q/%q not found", cleanScope, cleanKey)
		}
		return domain.AppSetting{}, fmt.Errorf("get setting %q/%q: %w", cleanScope, cleanKey, err)
	}
	return item, nil
}

func (s *Store) UpsertSetting(setting domain.AppSetting) (domain.AppSetting, error) {
	if err := requireStore(s); err != nil {
		return domain.AppSetting{}, err
	}
	setting.Scope = strings.TrimSpace(setting.Scope)
	setting.Key = strings.TrimSpace(setting.Key)
	if setting.Scope == "" || setting.Key == "" {
		return domain.AppSetting{}, fmt.Errorf("setting scope and key must not be empty")
	}
	value, err := jsonbOrEmptyObject(setting.Value)
	if err != nil {
		return domain.AppSetting{}, err
	}
	setting.UpdatedAt = now()
	_, err = s.pool.Exec(context.Background(), `
INSERT INTO settings(key, value, description, updated_at)
VALUES ($1,$2,$3,$4)
ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value, description=EXCLUDED.description, updated_at=EXCLUDED.updated_at`, settingStorageKey(setting.Scope, setting.Key), value, setting.Scope, setting.UpdatedAt)
	if err != nil {
		return domain.AppSetting{}, fmt.Errorf("upsert setting %q/%q: %w", setting.Scope, setting.Key, err)
	}
	return setting, nil
}

func (s *Store) ListSettings(scope string) ([]domain.AppSetting, error) {
	if err := requireStore(s); err != nil {
		return nil, err
	}
	cleanScope := strings.TrimSpace(scope)
	query := `SELECT key, value, updated_at FROM settings`
	args := make([]any, 0, 1)
	if cleanScope != "" {
		query += ` WHERE description=$1`
		args = append(args, cleanScope)
	}
	query += ` ORDER BY key ASC`
	rows, err := s.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AppSetting, 0)
	for rows.Next() {
		item, err := scanSetting(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate settings: %w", err)
	}
	return items, nil
}

func settingStorageKey(scope, key string) string {
	return strings.TrimSpace(scope) + ":" + strings.TrimSpace(key)
}

type settingScanner interface{ Scan(dest ...any) error }

func scanSetting(scanner settingScanner) (domain.AppSetting, error) {
	var storageKey string
	var value []byte
	var item domain.AppSetting
	if err := scanner.Scan(&storageKey, &value, &item.UpdatedAt); err != nil {
		return domain.AppSetting{}, err
	}
	parts := strings.SplitN(storageKey, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return domain.AppSetting{}, fmt.Errorf("stored setting key %q is invalid", storageKey)
	}
	parsedValue, err := unmarshalJSONB[map[string]any](value)
	if err != nil {
		return domain.AppSetting{}, err
	}
	item.Scope = parts[0]
	item.Key = parts[1]
	item.Value = parsedValue
	return item, nil
}
