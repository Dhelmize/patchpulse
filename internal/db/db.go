/**
 * Copyright (C) 2024 PatchPulse Contributors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	instance *sql.DB
	once     sync.Once
)

func Init(dbDir string) (*sql.DB, error) {
	var initErr error
	once.Do(func() {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create db directory: %w", err)
			return
		}

		dbPath := filepath.Join(dbDir, "patchpulse.db")
		dsn := fmt.Sprintf("%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", dbPath)

		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			initErr = fmt.Errorf("failed to open sqlite database: %w", err)
			return
		}

		// Single open connection for serialized safe SQLite writes
		db.SetMaxOpenConns(1)

		if err := migrate(db); err != nil {
			initErr = fmt.Errorf("database migration failed: %w", err)
			return
		}

		instance = db
	})

	return instance, initErr
}

func GetDB() *sql.DB {
	return instance
}

func Close() error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		source_type TEXT NOT NULL DEFAULT 'web',
		source_config TEXT NOT NULL DEFAULT '{}',
		adapter_type TEXT NOT NULL DEFAULT 'github-pr',
		adapter_config TEXT NOT NULL DEFAULT '{}',
		check_interval INTEGER NOT NULL DEFAULT 3600,
		last_hash TEXT,
		last_check INTEGER,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
	);

	CREATE TABLE IF NOT EXISTS patch_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id INTEGER NOT NULL,
		content_hash TEXT NOT NULL,
		raw_content TEXT NOT NULL,
		markdown_content TEXT NOT NULL,
		detected_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS agent_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target_id INTEGER,
		patch_history_id INTEGER,
		action_type TEXT NOT NULL,
		tool_calls TEXT,
		result TEXT,
		status TEXT NOT NULL,
		created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
		FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE SET NULL,
		FOREIGN KEY (patch_history_id) REFERENCES patch_history(id) ON DELETE SET NULL
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Dynamic column migration for upgrading existing sqlite databases
	_ = addColumnIfNotExists(db, "targets", "source_type", "TEXT NOT NULL DEFAULT 'web'")
	_ = addColumnIfNotExists(db, "targets", "source_config", "TEXT NOT NULL DEFAULT '{}'")

	return nil
}

func addColumnIfNotExists(db *sql.DB, table, column, colType string) error {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", table, column, colType)
	_, err := db.Exec(query)
	return err
}

// Target Queries

func GetAllTargets() ([]Target, error) {
	rows, err := instance.Query(`
		SELECT id, name, url, source_type, source_config, adapter_type, adapter_config,
		       check_interval, last_hash, last_check, enabled, created_at, updated_at
		FROM targets ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		if err := rows.Scan(
			&t.ID, &t.Name, &t.URL, &t.SourceType, &t.SourceConfig,
			&t.AdapterType, &t.AdapterConfig, &t.CheckInterval,
			&t.LastHash, &t.LastCheck, &t.Enabled, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func GetEnabledTargets() ([]Target, error) {
	rows, err := instance.Query(`
		SELECT id, name, url, source_type, source_config, adapter_type, adapter_config,
		       check_interval, last_hash, last_check, enabled, created_at, updated_at
		FROM targets WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []Target
	for rows.Next() {
		var t Target
		if err := rows.Scan(
			&t.ID, &t.Name, &t.URL, &t.SourceType, &t.SourceConfig,
			&t.AdapterType, &t.AdapterConfig, &t.CheckInterval,
			&t.LastHash, &t.LastCheck, &t.Enabled, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func GetTarget(id int64) (*Target, error) {
	var t Target
	err := instance.QueryRow(`
		SELECT id, name, url, source_type, source_config, adapter_type, adapter_config,
		       check_interval, last_hash, last_check, enabled, created_at, updated_at
		FROM targets WHERE id = ?`, id).Scan(
		&t.ID, &t.Name, &t.URL, &t.SourceType, &t.SourceConfig,
		&t.AdapterType, &t.AdapterConfig, &t.CheckInterval,
		&t.LastHash, &t.LastCheck, &t.Enabled, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTarget(t *Target) (int64, error) {
	now := time.Now().Unix()
	res, err := instance.Exec(`
		INSERT INTO targets (name, url, source_type, source_config, adapter_type, adapter_config, check_interval, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Name, t.URL, t.SourceType, t.SourceConfig, t.AdapterType, t.AdapterConfig, t.CheckInterval, t.Enabled, now, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateTarget(id int64, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().Unix()
	query := "UPDATE targets SET "
	var args []interface{}
	i := 0
	for col, val := range updates {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("%s = ?", col)
		args = append(args, val)
		i++
	}
	query += " WHERE id = ?"
	args = append(args, id)

	_, err := instance.Exec(query, args...)
	return err
}

func DeleteTarget(id int64) error {
	_, err := instance.Exec("DELETE FROM targets WHERE id = ?", id)
	return err
}

// Patch History Queries

func CreatePatchHistory(ph *PatchHistory) (int64, error) {
	now := time.Now().Unix()
	res, err := instance.Exec(`
		INSERT INTO patch_history (target_id, content_hash, raw_content, markdown_content, detected_at)
		VALUES (?, ?, ?, ?, ?)`,
		ph.TargetID, ph.ContentHash, ph.RawContent, ph.MarkdownContent, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetRecentPatchHistory(limit int) ([]PatchHistory, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := instance.Query(`
		SELECT id, target_id, content_hash, raw_content, markdown_content, detected_at
		FROM patch_history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patches []PatchHistory
	for rows.Next() {
		var p PatchHistory
		if err := rows.Scan(&p.ID, &p.TargetID, &p.ContentHash, &p.RawContent, &p.MarkdownContent, &p.DetectedAt); err != nil {
			return nil, err
		}
		patches = append(patches, p)
	}
	return patches, nil
}

func GetPatchHistoryByTarget(targetID int64, limit int) ([]PatchHistory, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := instance.Query(`
		SELECT id, target_id, content_hash, raw_content, markdown_content, detected_at
		FROM patch_history WHERE target_id = ? ORDER BY id DESC LIMIT ?`, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patches []PatchHistory
	for rows.Next() {
		var p PatchHistory
		if err := rows.Scan(&p.ID, &p.TargetID, &p.ContentHash, &p.RawContent, &p.MarkdownContent, &p.DetectedAt); err != nil {
			return nil, err
		}
		patches = append(patches, p)
	}
	return patches, nil
}

// Agent Log Queries

func CreateAgentLog(log *AgentLog) (int64, error) {
	now := time.Now().Unix()
	res, err := instance.Exec(`
		INSERT INTO agent_logs (target_id, patch_history_id, action_type, tool_calls, result, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		log.TargetID, log.PatchHistoryID, log.ActionType, log.ToolCalls, log.Result, log.Status, now,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetRecentAgentLogs(limit int) ([]AgentLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := instance.Query(`
		SELECT id, target_id, patch_history_id, action_type, tool_calls, result, status, created_at
		FROM agent_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AgentLog
	for rows.Next() {
		var l AgentLog
		if err := rows.Scan(&l.ID, &l.TargetID, &l.PatchHistoryID, &l.ActionType, &l.ToolCalls, &l.Result, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func GetAgentLogsByTarget(targetID int64, limit int) ([]AgentLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := instance.Query(`
		SELECT id, target_id, patch_history_id, action_type, tool_calls, result, status, created_at
		FROM agent_logs WHERE target_id = ? ORDER BY id DESC LIMIT ?`, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AgentLog
	for rows.Next() {
		var l AgentLog
		if err := rows.Scan(&l.ID, &l.TargetID, &l.PatchHistoryID, &l.ActionType, &l.ToolCalls, &l.Result, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// Global Settings Queries

func GetSetting(key string) (string, error) {
	var val string
	err := instance.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func GetAllSettings() (map[string]string, error) {
	rows, err := instance.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings[k] = v
	}
	return settings, nil
}

func SetSetting(key, value string) error {
	_, err := instance.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func SetSettings(settings map[string]string) error {
	tx, err := instance.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range settings {
		if _, err := stmt.Exec(k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}

