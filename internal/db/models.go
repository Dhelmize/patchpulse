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

type Target struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	URL           string  `json:"url"`
	SourceType    string  `json:"source_type"`
	SourceConfig  string  `json:"source_config"`
	AdapterType   string  `json:"adapter_type"`
	AdapterConfig string  `json:"adapter_config"`
	CheckInterval int     `json:"check_interval"`
	LastHash      *string `json:"last_hash"`
	LastCheck     *int64  `json:"last_check"`
	Enabled       int     `json:"enabled"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

type PatchHistory struct {
	ID              int64  `json:"id"`
	TargetID        int64  `json:"target_id"`
	ContentHash     string `json:"content_hash"`
	RawContent      string `json:"raw_content"`
	MarkdownContent string `json:"markdown_content"`
	DetectedAt      int64  `json:"detected_at"`
}

type AgentLog struct {
	ID             int64   `json:"id"`
	TargetID       *int64  `json:"target_id"`
	PatchHistoryID *int64  `json:"patch_history_id"`
	ActionType     string  `json:"action_type"`
	ToolCalls      *string `json:"tool_calls"`
	Result         *string `json:"result"`
	Status         string  `json:"status"`
	CreatedAt      int64   `json:"created_at"`
}

