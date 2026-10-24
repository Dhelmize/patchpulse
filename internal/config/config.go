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

package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	Host                string
	DBDir               string
	LLMProvider         string
	LLMAPIKey           string
	LLMModel            string
	LLMBaseURL          string
	GitHubToken         string
	GitHubDefaultBranch string
	MediaWikiAPIURL     string
	MediaWikiUsername   string
	MediaWikiPassword   string
	MediaWikiAPIToken   string
	DiscordWebhookURL   string
	DiscordBotToken     string
	UserAgent           string
	RequestTimeout      int
	MaxRetries          int
}

func Load() *Config {
	_ = godotenv.Load()

	dbDir := os.Getenv("DB_DIR")
	if dbDir == "" {
		configDir, err := os.UserConfigDir()
		if err == nil {
			dbDir = filepath.Join(configDir, "PatchPulse")
		} else {
			dbDir = "./data"
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	llmProvider := os.Getenv("LLM_PROVIDER")
	if llmProvider == "" {
		llmProvider = "openai"
	}

	llmModel := os.Getenv("LLM_MODEL")
	if llmModel == "" {
		if llmProvider == "gemini" {
			llmModel = "gemini-1.5-pro"
		} else {
			llmModel = "gpt-4"
		}
	}

	branch := os.Getenv("GITHUB_DEFAULT_BRANCH")
	if branch == "" {
		branch = "main"
	}

	userAgent := os.Getenv("USER_AGENT")
	if userAgent == "" {
		userAgent = "PatchPulse/1.0"
	}

	timeout := 30
	if toStr := os.Getenv("REQUEST_TIMEOUT"); toStr != "" {
		if v, err := strconv.Atoi(toStr); err == nil && v > 0 {
			timeout = v
		}
	}

	retries := 3
	if retStr := os.Getenv("MAX_RETRIES"); retStr != "" {
		if v, err := strconv.Atoi(retStr); err == nil && v >= 0 {
			retries = v
		}
	}

	return &Config{
		Port:                port,
		Host:                host,
		DBDir:               dbDir,
		LLMProvider:         llmProvider,
		LLMAPIKey:           os.Getenv("LLM_API_KEY"),
		LLMModel:            llmModel,
		LLMBaseURL:          os.Getenv("LLM_BASE_URL"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		GitHubDefaultBranch: branch,
		MediaWikiAPIURL:     os.Getenv("MEDIAWIKI_API_URL"),
		MediaWikiUsername:   os.Getenv("MEDIAWIKI_USERNAME"),
		MediaWikiPassword:   os.Getenv("MEDIAWIKI_PASSWORD"),
		MediaWikiAPIToken:   os.Getenv("MEDIAWIKI_API_TOKEN"),
		DiscordWebhookURL:   os.Getenv("DISCORD_WEBHOOK_URL"),
		DiscordBotToken:     os.Getenv("DISCORD_BOT_TOKEN"),
		UserAgent:           userAgent,
		RequestTimeout:      timeout,
		MaxRetries:          retries,
	}
}

// Resolve returns a new Config with values from the database settings table taking priority.
// If a database setting is empty, it falls back to the environment variable / config default.
func (c *Config) Resolve(dbSettings map[string]string) *Config {
	resolved := *c

	if v, ok := dbSettings["LLM_PROVIDER"]; ok && v != "" {
		resolved.LLMProvider = v
	}
	if v, ok := dbSettings["LLM_API_KEY"]; ok && v != "" {
		resolved.LLMAPIKey = v
	}
	if v, ok := dbSettings["LLM_MODEL"]; ok && v != "" {
		resolved.LLMModel = v
	}
	if v, ok := dbSettings["LLM_BASE_URL"]; ok && v != "" {
		resolved.LLMBaseURL = v
	}
	if v, ok := dbSettings["GITHUB_TOKEN"]; ok && v != "" {
		resolved.GitHubToken = v
	}
	if v, ok := dbSettings["GITHUB_DEFAULT_BRANCH"]; ok && v != "" {
		resolved.GitHubDefaultBranch = v
	}
	if v, ok := dbSettings["MEDIAWIKI_API_URL"]; ok && v != "" {
		resolved.MediaWikiAPIURL = v
	}
	if v, ok := dbSettings["MEDIAWIKI_USERNAME"]; ok && v != "" {
		resolved.MediaWikiUsername = v
	}
	if v, ok := dbSettings["MEDIAWIKI_PASSWORD"]; ok && v != "" {
		resolved.MediaWikiPassword = v
	}
	if v, ok := dbSettings["MEDIAWIKI_API_TOKEN"]; ok && v != "" {
		resolved.MediaWikiAPIToken = v
	}
	if v, ok := dbSettings["DISCORD_WEBHOOK_URL"]; ok && v != "" {
		resolved.DiscordWebhookURL = v
	}
	if v, ok := dbSettings["DISCORD_BOT_TOKEN"]; ok && v != "" {
		resolved.DiscordBotToken = v
	}
	if v, ok := dbSettings["USER_AGENT"]; ok && v != "" {
		resolved.UserAgent = v
	}

	return &resolved
}

