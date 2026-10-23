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
import { getDatabase } from './connection.js';

export interface Target {
  id?: number;
  name: string;
  url: string;
  adapter_type: string;
  adapter_config: string;
  check_interval: number;
  last_hash?: string;
  last_check?: number;
  enabled: number;
  created_at?: number;
  updated_at?: number;
}

export interface PatchHistory {
  id?: number;
  target_id: number;
  content_hash: string;
  raw_content: string;
  markdown_content: string;
  detected_at?: number;
}

export interface AgentLog {
  id?: number;
  target_id: number;
  patch_history_id?: number;
  action_type: string;
  tool_calls?: string;
  result?: string;
  status: string;
  created_at?: number;
}

// Target queries
export function createTarget(target: Omit<Target, 'id' | 'created_at' | 'updated_at'>): number {
  const db = getDatabase();
  const stmt = db.prepare(`
    INSERT INTO targets (name, url, adapter_type, adapter_config, check_interval, last_hash, last_check, enabled)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
  `);
  const result = stmt.run(
    target.name,
    target.url,
    target.adapter_type,
    target.adapter_config,
    target.check_interval,
    target.last_hash || null,
    target.last_check || null,
    target.enabled
  );
  return result.lastInsertRowid as number;
}

export function getTarget(id: number): Target | undefined {
  const db = getDatabase();
  const stmt = db.prepare('SELECT * FROM targets WHERE id = ?');
  return stmt.get(id) as Target | undefined;
}

export function getAllTargets(): Target[] {
  const db = getDatabase();
  const stmt = db.prepare('SELECT * FROM targets ORDER BY created_at DESC');
  return stmt.all() as Target[];
}

export function getEnabledTargets(): Target[] {
  const db = getDatabase();
  const stmt = db.prepare('SELECT * FROM targets WHERE enabled = 1 ORDER BY created_at DESC');
  return stmt.all() as Target[];
}

export function updateTarget(id: number, updates: Partial<Omit<Target, 'id' | 'created_at'>>): void {
  const db = getDatabase();
  const fields: string[] = [];
  const values: any[] = [];

  if (updates.name !== undefined) {
    fields.push('name = ?');
    values.push(updates.name);
  }
  if (updates.url !== undefined) {
    fields.push('url = ?');
    values.push(updates.url);
  }
  if (updates.adapter_type !== undefined) {
    fields.push('adapter_type = ?');
    values.push(updates.adapter_type);
  }
  if (updates.adapter_config !== undefined) {
    fields.push('adapter_config = ?');
    values.push(updates.adapter_config);
  }
  if (updates.check_interval !== undefined) {
    fields.push('check_interval = ?');
    values.push(updates.check_interval);
  }
  if (updates.last_hash !== undefined) {
    fields.push('last_hash = ?');
    values.push(updates.last_hash);
  }
  if (updates.last_check !== undefined) {
    fields.push('last_check = ?');
    values.push(updates.last_check);
  }
  if (updates.enabled !== undefined) {
    fields.push('enabled = ?');
    values.push(updates.enabled);
  }

  fields.push('updated_at = strftime(\'%s\', \'now\')');
  values.push(id);

  const stmt = db.prepare(`UPDATE targets SET ${fields.join(', ')} WHERE id = ?`);
  stmt.run(...values);
}

export function deleteTarget(id: number): void {
  const db = getDatabase();
  const stmt = db.prepare('DELETE FROM targets WHERE id = ?');
  stmt.run(id);
}

// Patch history queries
export function createPatchHistory(patch: Omit<PatchHistory, 'id' | 'detected_at'>): number {
  const db = getDatabase();
  const stmt = db.prepare(`
    INSERT INTO patch_history (target_id, content_hash, raw_content, markdown_content)
    VALUES (?, ?, ?, ?)
  `);
  const result = stmt.run(
    patch.target_id,
    patch.content_hash,
    patch.raw_content,
    patch.markdown_content
  );
  return result.lastInsertRowid as number;
}

export function getPatchHistory(id: number): PatchHistory | undefined {
  const db = getDatabase();
  const stmt = db.prepare('SELECT * FROM patch_history WHERE id = ?');
  return stmt.get(id) as PatchHistory | undefined;
}

export function getPatchHistoryByTarget(targetId: number, limit: number = 10): PatchHistory[] {
  const db = getDatabase();
  const stmt = db.prepare(`
    SELECT * FROM patch_history
    WHERE target_id = ?
    ORDER BY detected_at DESC
    LIMIT ?
  `);
  return stmt.all(targetId, limit) as PatchHistory[];
}

export function getRecentPatchHistory(limit: number = 20): PatchHistory[] {
  const db = getDatabase();
  const stmt = db.prepare(`
    SELECT * FROM patch_history
    ORDER BY detected_at DESC
    LIMIT ?
  `);
  return stmt.all(limit) as PatchHistory[];
}

// Agent log queries
export function createAgentLog(log: Omit<AgentLog, 'id' | 'created_at'>): number {
  const db = getDatabase();
  const stmt = db.prepare(`
    INSERT INTO agent_logs (target_id, patch_history_id, action_type, tool_calls, result, status)
    VALUES (?, ?, ?, ?, ?, ?)
  `);
  const result = stmt.run(
    log.target_id,
    log.patch_history_id || null,
    log.action_type,
    log.tool_calls || null,
    log.result || null,
    log.status
  );
  return result.lastInsertRowid as number;
}

export function getAgentLogsByTarget(targetId: number, limit: number = 20): AgentLog[] {
  const db = getDatabase();
  const stmt = db.prepare(`
    SELECT * FROM agent_logs
    WHERE target_id = ?
    ORDER BY created_at DESC
    LIMIT ?
  `);
  return stmt.all(targetId, limit) as AgentLog[];
}

export function getAgentLogsByPatch(patchHistoryId: number): AgentLog[] {
  const db = getDatabase();
  const stmt = db.prepare(`
    SELECT * FROM agent_logs
    WHERE patch_history_id = ?
    ORDER BY created_at ASC
  `);
  return stmt.all(patchHistoryId) as AgentLog[];
}

export function getRecentAgentLogs(limit: number = 50): AgentLog[] {
  const db = getDatabase();
  const stmt = db.prepare(`
    SELECT * FROM agent_logs
    ORDER BY created_at DESC
    LIMIT ?
  `);
  return stmt.all(limit) as AgentLog[];
}
