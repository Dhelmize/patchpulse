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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GitHubConfig struct {
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	Token      string `json:"token"`
	BaseBranch string `json:"base_branch"`
}

type GitHubPRAdapter struct {
	owner       string
	repo        string
	token       string
	baseBranch  string
	client      *http.Client
	stagedEdits map[string]string
	diffURL     string
}

func NewGitHubPRAdapter(cfg GitHubConfig) *GitHubPRAdapter {
	branch := cfg.BaseBranch
	if branch == "" {
		branch = "main"
	}
	return &GitHubPRAdapter{
		owner:       cfg.Owner,
		repo:        cfg.Repo,
		token:       cfg.Token,
		baseBranch:  branch,
		client:      &http.Client{Timeout: 30 * time.Second},
		stagedEdits: make(map[string]string),
	}
}

func (a *GitHubPRAdapter) Initialize(ctx context.Context) error {
	if a.owner == "" || a.repo == "" {
		return fmt.Errorf("GitHub owner and repo are required")
	}
	return nil
}

func (a *GitHubPRAdapter) GetType() string {
	return "github-pr"
}

func (a *GitHubPRAdapter) IsNotificationOnly() bool {
	return false
}

func (a *GitHubPRAdapter) GetDiffURL() string {
	return a.diffURL
}

func (a *GitHubPRAdapter) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	url := fmt.Sprintf("https://api.github.com%s", endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "PatchPulse/1.0")
	if a.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return a.client.Do(req)
}

func (a *GitHubPRAdapter) ReadFile(ctx context.Context, path string) (string, error) {
	if content, exists := a.stagedEdits[path]; exists {
		return content, nil
	}

	endpoint := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", a.owner, a.repo, strings.TrimPrefix(path, "/"), a.baseBranch)
	resp, err := a.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found: %s", path)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github api error (%d): %s", resp.StatusCode, string(b))
	}

	var res struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res.Encoding == "base64" {
		cleaned := strings.ReplaceAll(res.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 file content: %w", err)
		}
		return string(decoded), nil
	}

	return res.Content, nil
}

func (a *GitHubPRAdapter) SearchRepo(ctx context.Context, query string) ([]string, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/git/trees/%s?recursive=1", a.owner, a.repo, a.baseBranch)
	resp, err := a.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch tree: status %d", resp.StatusCode)
	}

	var res struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var results []string
	qLower := strings.ToLower(query)
	for _, item := range res.Tree {
		if item.Type == "blob" {
			if query == "" || strings.Contains(strings.ToLower(item.Path), qLower) {
				results = append(results, item.Path)
			}
		}
	}
	return results, nil
}

func (a *GitHubPRAdapter) UpdateFile(ctx context.Context, path, content, commitMessage string) error {
	a.stagedEdits[strings.TrimPrefix(path, "/")] = content
	return nil
}

func (a *GitHubPRAdapter) CreatePullRequest(ctx context.Context, title, body, branch string) (string, error) {
	if len(a.stagedEdits) == 0 {
		return "", fmt.Errorf("no files modified to create PR")
	}

	if branch == "" {
		branch = fmt.Sprintf("patchpulse-%d", time.Now().Unix())
	}

	// 1. Get base branch commit SHA
	refResp, err := a.doRequest(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/git/ref/heads/%s", a.owner, a.repo, a.baseBranch), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get base branch ref: %w", err)
	}
	defer refResp.Body.Close()

	if refResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get base branch %s: status %d", a.baseBranch, refResp.StatusCode)
	}

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(refResp.Body).Decode(&refData); err != nil {
		return "", err
	}
	baseSHA := refData.Object.SHA

	// 2. Create tree entries for all staged files
	type treeItem struct {
		Path    string `json:"path"`
		Mode    string `json:"mode"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}

	var treeEntries []treeItem
	for p, c := range a.stagedEdits {
		treeEntries = append(treeEntries, treeItem{
			Path:    p,
			Mode:    "100644",
			Type:    "blob",
			Content: c,
		})
	}

	treeResp, err := a.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/git/trees", a.owner, a.repo), map[string]interface{}{
		"base_tree": baseSHA,
		"tree":      treeEntries,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create tree: %w", err)
	}
	defer treeResp.Body.Close()

	var treeData struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(treeResp.Body).Decode(&treeData); err != nil {
		return "", err
	}

	// 3. Create commit
	commitResp, err := a.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/git/commits", a.owner, a.repo), map[string]interface{}{
		"message": title,
		"tree":    treeData.SHA,
		"parents": []string{baseSHA},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create commit: %w", err)
	}
	defer commitResp.Body.Close()

	var commitData struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(commitResp.Body).Decode(&commitData); err != nil {
		return "", err
	}

	// 4. Create new branch ref pointing to new commit
	newRefResp, err := a.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/git/refs", a.owner, a.repo), map[string]interface{}{
		"ref": fmt.Sprintf("refs/heads/%s", branch),
		"sha": commitData.SHA,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create branch ref: %w", err)
	}
	defer newRefResp.Body.Close()

	// 5. Open pull request
	prResp, err := a.doRequest(ctx, http.MethodPost, fmt.Sprintf("/repos/%s/%s/pulls", a.owner, a.repo), map[string]interface{}{
		"title": title,
		"body":  body,
		"head":  branch,
		"base":  a.baseBranch,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}
	defer prResp.Body.Close()

	var prData struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(prResp.Body).Decode(&prData); err != nil {
		return "", err
	}

	a.diffURL = prData.HTMLURL
	a.stagedEdits = make(map[string]string) // clear staged edits

	return prData.HTMLURL, nil
}

func (a *GitHubPRAdapter) SendNotification(ctx context.Context, message, diffURL string) error {
	return nil
}

