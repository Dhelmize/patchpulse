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
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MediaWikiConfig struct {
	APIURL   string `json:"api_url"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIToken string `json:"api_token"`
}

type MediaWikiAdapter struct {
	apiURL      string
	username    string
	password    string
	apiToken    string
	authToken   string
	client      *http.Client
	stagedEdits map[string]string
	diffURL     string
}

func NewMediaWikiAdapter(cfg MediaWikiConfig) *MediaWikiAdapter {
	return &MediaWikiAdapter{
		apiURL:      cfg.APIURL,
		username:    cfg.Username,
		password:    cfg.Password,
		apiToken:    cfg.APIToken,
		client:      &http.Client{Timeout: 30 * time.Second},
		stagedEdits: make(map[string]string),
	}
}

func (a *MediaWikiAdapter) Initialize(ctx context.Context) error {
	if a.apiURL == "" {
		return fmt.Errorf("MediaWiki API URL is required")
	}

	if a.apiToken != "" {
		a.authToken = a.apiToken
		return nil
	}

	if a.username != "" && a.password != "" {
		// 1. Get login token
		u := fmt.Sprintf("%s?action=query&meta=tokens&type=login&format=json", a.apiURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		resp, err := a.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var tokenRes struct {
			Query struct {
				Tokens struct {
					LoginToken string `json:"logintoken"`
				} `json:"tokens"`
			} `json:"query"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
			return err
		}

		// 2. Client login
		form := url.Values{}
		form.Set("action", "clientlogin")
		form.Set("username", a.username)
		form.Set("password", a.password)
		form.Set("logintoken", tokenRes.Query.Tokens.LoginToken)
		form.Set("loginreturnurl", a.apiURL)
		form.Set("format", "json")

		loginReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		loginResp, err := a.client.Do(loginReq)
		if err != nil {
			return err
		}
		defer loginResp.Body.Close()

		// 3. Fetch CSRF token for edits
		csrfReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s?action=query&meta=tokens&type=csrf&format=json", a.apiURL), nil)
		if err == nil {
			csrfResp, err := a.client.Do(csrfReq)
			if err == nil {
				defer csrfResp.Body.Close()
				var csrfRes struct {
					Query struct {
						Tokens struct {
							CSRFToken string `json:"csrftoken"`
						} `json:"tokens"`
					} `json:"query"`
				}
				if json.NewDecoder(csrfResp.Body).Decode(&csrfRes) == nil {
					a.authToken = csrfRes.Query.Tokens.CSRFToken
				}
			}
		}
	}

	return nil
}

func (a *MediaWikiAdapter) GetType() string {
	return "mediawiki"
}

func (a *MediaWikiAdapter) IsNotificationOnly() bool {
	return false
}

func (a *MediaWikiAdapter) GetDiffURL() string {
	return a.diffURL
}

func (a *MediaWikiAdapter) ReadFile(ctx context.Context, title string) (string, error) {
	if content, exists := a.stagedEdits[title]; exists {
		return content, nil
	}

	u := fmt.Sprintf("%s?action=query&prop=revisions&titles=%s&rvprop=content&rvslots=main&format=json", a.apiURL, url.QueryEscape(title))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Query struct {
			Pages map[string]struct {
				Revisions []struct {
					Slots struct {
						Main struct {
							Content string `json:"*"`
						} `json:"main"`
					} `json:"slots"`
				} `json:"revisions"`
			} `json:"pages"`
		} `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	for _, page := range res.Query.Pages {
		if len(page.Revisions) > 0 {
			return page.Revisions[0].Slots.Main.Content, nil
		}
	}

	return "", fmt.Errorf("page not found or content empty: %s", title)
}

func (a *MediaWikiAdapter) SearchRepo(ctx context.Context, query string) ([]string, error) {
	u := fmt.Sprintf("%s?action=query&list=search&srsearch=%s&srlimit=20&format=json", a.apiURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	var titles []string
	for _, item := range res.Query.Search {
		titles = append(titles, item.Title)
	}
	return titles, nil
}

func (a *MediaWikiAdapter) UpdateFile(ctx context.Context, path, content, commitMessage string) error {
	a.stagedEdits[path] = content
	return nil
}

func (a *MediaWikiAdapter) CreatePullRequest(ctx context.Context, title, body, branch string) (string, error) {
	if len(a.stagedEdits) == 0 {
		return "", fmt.Errorf("no edits staged to apply")
	}

	var firstDiff string
	for pageTitle, content := range a.stagedEdits {
		form := url.Values{}
		form.Set("action", "edit")
		form.Set("title", pageTitle)
		form.Set("text", content)
		form.Set("summary", title)
		form.Set("format", "json")
		if a.authToken != "" {
			form.Set("token", a.authToken)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.apiURL, strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := a.client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var editRes struct {
			Edit struct {
				Result string `json:"result"`
				NewRev int64  `json:"newrevid"`
				OldRev int64  `json:"oldrevid"`
			} `json:"edit"`
			Error *struct {
				Info string `json:"info"`
			} `json:"error"`
		}
		b, _ := io.ReadAll(resp.Body)
		_ = json.Unmarshal(b, &editRes)

		if editRes.Error != nil {
			return "", fmt.Errorf("edit failed for %s: %s", pageTitle, editRes.Error.Info)
		}

		indexURL := strings.Replace(a.apiURL, "api.php", "index.php", 1)
		diffURL := fmt.Sprintf("%s?title=%s&diff=%d&oldid=%d", indexURL, url.QueryEscape(pageTitle), editRes.Edit.NewRev, editRes.Edit.OldRev)
		if firstDiff == "" {
			firstDiff = diffURL
		}
	}

	a.diffURL = firstDiff
	a.stagedEdits = make(map[string]string)
	return firstDiff, nil
}

func (a *MediaWikiAdapter) SendNotification(ctx context.Context, message, diffURL string) error {
	return nil
}

