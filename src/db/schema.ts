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
import Database from 'better-sqlite3';

export function initializeSchema(db: Database.Database): void {
  // Targets table - stores watched URLs and their configurations
  db.exec(`
    CREATE TABLE IF NOT EXISTS targets (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      name TEXT NOT NULL,
      url TEXT NOT NULL UNIQUE,
      adapter_type TEXT NOT NULL,
      adapter_config TEXT NOT NULL,
      check_interval INTEGER NOT NULL DEFAULT 3600,
      last_hash TEXT,
      last_check INTEGER,
      enabled INTEGER NOT NULL DEFAULT 1,
      created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
      updated_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
    )
  `);

  // Patch history table - stores detected patch notes
  db.exec(`
    CREATE TABLE IF NOT EXISTS patch_history (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      target_id INTEGER NOT NULL,
      content_hash TEXT NOT NULL,
      raw_content TEXT NOT NULL,
      markdown_content TEXT NOT NULL,
      detected_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
      FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE
    )
  `);

  // Agent logs table - stores agent actions and tool calls
  db.exec(`
    CREATE TABLE IF NOT EXISTS agent_logs (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      target_id INTEGER NOT NULL,
      patch_history_id INTEGER,
      action_type TEXT NOT NULL,
      tool_calls TEXT,
      result TEXT,
      status TEXT NOT NULL,
      created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
      FOREIGN KEY (target_id) REFERENCES targets(id) ON DELETE CASCADE,
      FOREIGN KEY (patch_history_id) REFERENCES patch_history(id) ON DELETE CASCADE
    )
  `);

  // Create indexes for better query performance
  db.exec(`
    CREATE INDEX IF NOT EXISTS idx_targets_enabled ON targets(enabled);
    CREATE INDEX IF NOT EXISTS idx_patch_history_target ON patch_history(target_id);
    CREATE INDEX IF NOT EXISTS idx_agent_logs_target ON agent_logs(target_id);
    CREATE INDEX IF NOT EXISTS idx_agent_logs_patch ON agent_logs(patch_history_id);
  `);
}
