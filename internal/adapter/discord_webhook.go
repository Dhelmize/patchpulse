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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type DiscordWebhookAdapter struct {
	webhookURL string
	client     *http.Client
	diffURL    string
}

func NewDiscordWebhookAdapter(webhookURL string) *DiscordWebhookAdapter {
	return &DiscordWebhookAdapter{
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *DiscordWebhookAdapter) Initialize(ctx context.Context) error {
	if a.webhookURL == "" {
		return fmt.Errorf("Discord Webhook URL is required")
	}
	return nil
}

func (a *DiscordWebhookAdapter) GetType() string {
	return "discord-webhook"
}

func (a *DiscordWebhookAdapter) IsNotificationOnly() bool {
	return true
}

func (a *DiscordWebhookAdapter) GetDiffURL() string {
	return a.diffURL
}

func (a *DiscordWebhookAdapter) ReadFile(ctx context.Context, path string) (string, error) {
	return "", fmt.Errorf("ReadFile is not supported on notification-only Discord webhook adapter")
}

func (a *DiscordWebhookAdapter) SearchRepo(ctx context.Context, query string) ([]string, error) {
	return nil, fmt.Errorf("SearchRepo is not supported on notification-only Discord webhook adapter")
}

func (a *DiscordWebhookAdapter) UpdateFile(ctx context.Context, path, content, commitMessage string) error {
	return nil
}

func (a *DiscordWebhookAdapter) CreatePullRequest(ctx context.Context, title, body, branch string) (string, error) {
	return "", nil
}

func (a *DiscordWebhookAdapter) SendNotification(ctx context.Context, message, diffURL string) error {
	if a.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload := map[string]interface{}{
		"content": message,
	}

	if diffURL != "" {
		payload["embeds"] = []map[string]interface{}{
			{
				"title":       "Patch Details & Diff",
				"url":         diffURL,
				"color":       3866870, // #3B82F6 (blue)
				"description": "Click the link above to view the full changelog and generated diff.",
				"footer": map[string]interface{}{
					"text": "PatchPulse Agentic Pipeline",
				},
			},
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook responded with HTTP %d", resp.StatusCode)
	}

	a.diffURL = diffURL
	return nil
}

