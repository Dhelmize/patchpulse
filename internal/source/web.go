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
	"fmt"
	"net/http"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"patchpulse/internal/config"
)

type WebSource struct {
	url       string
	userAgent string
	timeout   time.Duration
	client    *http.Client
}

func NewWebSource(url string, cfg *config.Config) *WebSource {
	timeout := 30 * time.Second
	ua := "PatchPulse/1.0"
	if cfg != nil {
		if cfg.RequestTimeout > 0 {
			timeout = time.Duration(cfg.RequestTimeout) * time.Second
		}
		if cfg.UserAgent != "" {
			ua = cfg.UserAgent
		}
	}

	return &WebSource{
		url:       url,
		userAgent: ua,
		timeout:   timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *WebSource) Fetch(ctx context.Context) (*ContentResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	title := strings.TrimSpace(doc.Find("title").Text())

	// Strip out non-content tags
	doc.Find("script, style, nav, footer, header, noscript, iframe, svg, [role='navigation'], [role='banner']").Remove()

	// Prioritize main article/content container
	var contentHtml string
	selectors := []string{
		"article",
		"main",
		"[role='main']",
		".patch-notes",
		".changelog",
		"#content",
		".content",
		"#main-content",
		"body",
	}

	for _, sel := range selectors {
		selMatch := doc.Find(sel)
		if selMatch.Length() > 0 {
			h, err := selMatch.First().Html()
			if err == nil && len(strings.TrimSpace(h)) > 0 {
				contentHtml = h
				break
			}
		}
	}

	if contentHtml == "" {
		contentHtml, _ = doc.Find("body").Html()
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(contentHtml)
	if err != nil {
		markdown = strings.TrimSpace(doc.Text())
	}

	hash := ComputeSHA256(markdown)

	return &ContentResult{
		RawContent:      contentHtml,
		MarkdownContent: markdown,
		Hash:            hash,
		Title:           title,
	}, nil
}

