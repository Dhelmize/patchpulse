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
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/mmcdole/gofeed"
	"patchpulse/internal/config"
)

type RSSSource struct {
	url    string
	cfg    *config.Config
	parser *gofeed.Parser
}

func NewRSSSource(url string, cfg *config.Config) *RSSSource {
	fp := gofeed.NewParser()
	if cfg != nil && cfg.UserAgent != "" {
		fp.UserAgent = cfg.UserAgent
	}
	return &RSSSource{
		url:    url,
		cfg:    cfg,
		parser: fp,
	}
}

func (s *RSSSource) Fetch(ctx context.Context) (*ContentResult, error) {
	feed, err := s.parser.ParseURLWithContext(s.url, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in RSS feed")
	}

	latest := feed.Items[0]
	raw := latest.Content
	if raw == "" {
		raw = latest.Description
	}
	if raw == "" {
		raw = latest.Title
	}

	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(raw)
	if err != nil {
		markdown = raw
	}

	title := latest.Title
	if title == "" {
		title = feed.Title
	}

	builder := strings.Builder{}
	if title != "" {
		builder.WriteString(fmt.Sprintf("# %s\n\n", title))
	}
	if latest.Link != "" {
		builder.WriteString(fmt.Sprintf("Link: %s\n\n", latest.Link))
	}
	builder.WriteString(markdown)

	fullMarkdown := builder.String()
	hash := ComputeSHA256(fullMarkdown)

	return &ContentResult{
		RawContent:      raw,
		MarkdownContent: fullMarkdown,
		Hash:            hash,
		Title:           title,
	}, nil
}

