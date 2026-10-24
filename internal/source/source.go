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

package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"patchpulse/internal/config"
)

type ContentResult struct {
	RawContent      string
	MarkdownContent string
	Hash            string
	Title           string
}

type Source interface {
	Fetch(ctx context.Context) (*ContentResult, error)
}

func ComputeSHA256(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

func NewSource(sourceType, sourceConfig, fallbackURL string, cfg *config.Config) (Source, error) {
	switch strings.ToLower(sourceType) {
	case "rss":
		url := fallbackURL
		if sourceConfig != "" && sourceConfig != "{}" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(sourceConfig), &m); err == nil {
				if u, ok := m["url"].(string); ok && u != "" {
					url = u
				}
			}
		}
		if url == "" {
			return nil, fmt.Errorf("missing RSS URL")
		}
		return NewRSSSource(url, cfg), nil

	case "discord-channel", "discord":
		channelID := ""
		botToken := cfg.DiscordBotToken
		if sourceConfig != "" && sourceConfig != "{}" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(sourceConfig), &m); err == nil {
				if cid, ok := m["channel_id"].(string); ok && cid != "" {
					channelID = cid
				}
				if bt, ok := m["bot_token"].(string); ok && bt != "" {
					botToken = bt
				}
			}
		}
		if channelID == "" {
			// check if fallbackURL contains channel ID
			channelID = strings.TrimSpace(fallbackURL)
		}
		if channelID == "" {
			return nil, fmt.Errorf("missing Discord channel_id")
		}
		return NewDiscordChannelSource(channelID, botToken), nil

	case "web", "":
		url := fallbackURL
		if sourceConfig != "" && sourceConfig != "{}" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(sourceConfig), &m); err == nil {
				if u, ok := m["url"].(string); ok && u != "" {
					url = u
				}
			}
		}
		if url == "" {
			return nil, fmt.Errorf("missing target URL")
		}
		return NewWebSource(url, cfg), nil

	default:
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
}

