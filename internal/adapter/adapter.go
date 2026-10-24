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

package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"patchpulse/internal/config"
)

type Adapter interface {
	Initialize(ctx context.Context) error
	GetType() string
	IsNotificationOnly() bool
	ReadFile(ctx context.Context, path string) (string, error)
	SearchRepo(ctx context.Context, query string) ([]string, error)
	UpdateFile(ctx context.Context, path, content, commitMessage string) error
	CreatePullRequest(ctx context.Context, title, body, branch string) (string, error)
	SendNotification(ctx context.Context, message, diffURL string) error
	GetDiffURL() string
}

func NewAdapter(adapterType, adapterConfig string, cfg *config.Config) (Adapter, error) {
	switch strings.ToLower(adapterType) {
	case "github-pr", "github":
		var ghCfg GitHubConfig
		if adapterConfig != "" && adapterConfig != "{}" {
			_ = json.Unmarshal([]byte(adapterConfig), &ghCfg)
		}
		if ghCfg.Owner == "" && ghCfg.Repo == "" {
			ghCfg.Owner = "Dhelmize"
			ghCfg.Repo = "patchpulse"
		}
		if ghCfg.Token == "" {
			ghCfg.Token = cfg.GitHubToken
		}
		if ghCfg.BaseBranch == "" {
			ghCfg.BaseBranch = cfg.GitHubDefaultBranch
		}
		return NewGitHubPRAdapter(ghCfg), nil

	case "mediawiki":
		var mwCfg MediaWikiConfig
		if adapterConfig != "" && adapterConfig != "{}" {
			_ = json.Unmarshal([]byte(adapterConfig), &mwCfg)
		}
		if mwCfg.APIURL == "" {
			mwCfg.APIURL = cfg.MediaWikiAPIURL
		}
		if mwCfg.Username == "" {
			mwCfg.Username = cfg.MediaWikiUsername
		}
		if mwCfg.Password == "" {
			mwCfg.Password = cfg.MediaWikiPassword
		}
		if mwCfg.APIToken == "" {
			mwCfg.APIToken = cfg.MediaWikiAPIToken
		}
		return NewMediaWikiAdapter(mwCfg), nil

	case "discord-webhook", "discord":
		webhookURL := cfg.DiscordWebhookURL
		if adapterConfig != "" && adapterConfig != "{}" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(adapterConfig), &m); err == nil {
				if u, ok := m["webhook_url"].(string); ok && u != "" {
					webhookURL = u
				}
			}
		}
		return NewDiscordWebhookAdapter(webhookURL), nil

	case "local-dir", "local":
		var localCfg LocalDirConfig
		if adapterConfig != "" && adapterConfig != "{}" {
			_ = json.Unmarshal([]byte(adapterConfig), &localCfg)
		}
		if localCfg.Directory == "" {
			localCfg.Directory = "./"
		}
		return NewLocalDirAdapter(localCfg), nil

	default:
		return nil, fmt.Errorf("unsupported adapter type: %s", adapterType)
	}
}

