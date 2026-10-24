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

	"github.com/bwmarrin/discordgo"
)

type DiscordChannelSource struct {
	channelID string
	botToken  string
}

func NewDiscordChannelSource(channelID, botToken string) *DiscordChannelSource {
	return &DiscordChannelSource{
		channelID: channelID,
		botToken:  botToken,
	}
}

func (s *DiscordChannelSource) Fetch(ctx context.Context) (*ContentResult, error) {
	if s.botToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is required to fetch Discord channel updates")
	}

	token := s.botToken
	if !strings.HasPrefix(token, "Bot ") {
		token = "Bot " + token
	}

	dg, err := discordgo.New(token)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discord client: %w", err)
	}

	messages, err := dg.ChannelMessages(s.channelID, 5, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch channel messages for %s: %w", s.channelID, err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages found in Discord channel %s", s.channelID)
	}

	var rawBuilder strings.Builder
	var mdBuilder strings.Builder

	// The newest message is messages[0]
	primaryMsg := messages[0]
	title := fmt.Sprintf("Discord Update (Msg ID: %s)", primaryMsg.ID)
	mdBuilder.WriteString(fmt.Sprintf("# Discord Update #%s\n\n", primaryMsg.ID))
	mdBuilder.WriteString(fmt.Sprintf("**Timestamp:** %s\n\n", primaryMsg.Timestamp.Format("2006-01-02 15:04:05 UTC")))

	for i, msg := range messages {
		if msg.Content != "" {
			rawBuilder.WriteString(msg.Content + "\n")
			mdBuilder.WriteString(msg.Content + "\n\n")
		}

		for _, embed := range msg.Embeds {
			if embed.Title != "" {
				rawBuilder.WriteString(embed.Title + "\n")
				mdBuilder.WriteString(fmt.Sprintf("### %s\n\n", embed.Title))
			}
			if embed.Description != "" {
				rawBuilder.WriteString(embed.Description + "\n")
				mdBuilder.WriteString(embed.Description + "\n\n")
			}
			for _, field := range embed.Fields {
				rawBuilder.WriteString(field.Name + ": " + field.Value + "\n")
				mdBuilder.WriteString(fmt.Sprintf("- **%s:** %s\n", field.Name, field.Value))
			}
		}

		// Only look at the latest message for hash/title if distinct, or combine recent
		if i == 0 {
			break
		}
	}

	rawContent := strings.TrimSpace(rawBuilder.String())
	markdownContent := strings.TrimSpace(mdBuilder.String())
	hash := ComputeSHA256(markdownContent)

	return &ContentResult{
		RawContent:      rawContent,
		MarkdownContent: markdownContent,
		Hash:            hash,
		Title:           title,
	}, nil
}

