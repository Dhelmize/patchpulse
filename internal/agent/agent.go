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

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"patchpulse/internal/adapter"
	"patchpulse/internal/config"
	"patchpulse/internal/db"
)

type Agent struct {
	provider LLMProvider
	adapter  adapter.Adapter
	cfg      *config.Config
}

func NewAgent(adp adapter.Adapter, cfg *config.Config) (*Agent, error) {
	provider, err := NewLLMProvider(cfg)
	if err != nil {
		return nil, err
	}

	return &Agent{
		provider: provider,
		adapter:  adp,
		cfg:      cfg,
	}, nil
}

func (a *Agent) ProcessPatch(ctx context.Context, targetID, patchID int64, patchNotes, targetInfo string) error {
	// If adapter is notification-only, summarize the diff and dispatch notification directly
	if a.adapter.IsNotificationOnly() {
		return a.processNotificationOnly(ctx, targetID, patchID, patchNotes, targetInfo)
	}

	return a.processWithToolCalling(ctx, targetID, patchID, patchNotes, targetInfo)
}

func (a *Agent) processNotificationOnly(ctx context.Context, targetID, patchID int64, patchNotes, targetInfo string) error {
	systemPrompt := "You are an automated technical documentation assistant. Generate a concise, well-structured changelog summary from the patch notes provided."
	userPrompt := fmt.Sprintf("Target Info: %s\n\nPatch Notes:\n%s\n\nProvide a high-level overview of notable changes, bug fixes, and feature adjustments.", targetInfo, patchNotes)

	msgs := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	res, err := a.provider.Generate(ctx, msgs, nil)
	if err != nil {
		return fmt.Errorf("failed to generate patch summary: %w", err)
	}

	summary := res.Content
	if summary == "" {
		summary = "New patch detected and processed."
	}

	// Send notification via adapter
	if err := a.adapter.SendNotification(ctx, summary, ""); err != nil {
		log.Printf("failed to send notification: %v", err)
	}

	logResult := "Notification dispatched to webhook"
	_, _ = db.CreateAgentLog(&db.AgentLog{
		TargetID:       &targetID,
		PatchHistoryID: &patchID,
		ActionType:     "send_notification",
		Result:         &logResult,
		Status:         "success",
	})

	return nil
}

func (a *Agent) processWithToolCalling(ctx context.Context, targetID, patchID int64, patchNotes, targetInfo string) error {
	systemPrompt := `You are PatchPulse, an expert agentic documentation and wiki maintenance assistant.
Analyze new game or software patch notes and determine what docs, stats, and guides need updating.
Use the provided tools:
1. search_repo(query): discover relevant documentation or data files.
2. read_file(path): inspect existing file contents.
3. patch_file(path, content, commit_message): propose and stage accurate file revisions.
4. create_pull_request(title, body): finalize changes once all needed updates are staged.

Be precise, preserve surrounding styling/syntax, and conclude by creating a pull request or finalizing the updates.`

	userPrompt := fmt.Sprintf("Target Details: %s\n\nNew Patch Notes:\n%s\n\nPlease search for relevant files, read their current contents, update them accordingly, and create a pull request.", targetInfo, patchNotes)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	tools := GetDefaultTools()
	maxIterations := 10
	prCreated := false

	for iteration := 0; iteration < maxIterations; iteration++ {
		response, err := a.provider.Generate(ctx, messages, tools)
		if err != nil {
			errStr := err.Error()
			_, _ = db.CreateAgentLog(&db.AgentLog{
				TargetID:       &targetID,
				PatchHistoryID: &patchID,
				ActionType:     "agent_iteration_error",
				Result:         &errStr,
				Status:         "error",
			})
			return fmt.Errorf("agent iteration error: %w", err)
		}

		messages = append(messages, *response)

		if len(response.ToolCalls) == 0 {
			// No further tools called, agent has finished speaking
			break
		}

		for _, tc := range response.ToolCalls {
			toolResult, isPR := a.executeTool(ctx, tc)
			if isPR {
				prCreated = true
			}

			tcBytes, _ := json.Marshal(tc)
			tcStr := string(tcBytes)
			_, _ = db.CreateAgentLog(&db.AgentLog{
				TargetID:       &targetID,
				PatchHistoryID: &patchID,
				ActionType:     tc.Name,
				ToolCalls:      &tcStr,
				Result:         &toolResult,
				Status:         "success",
			})

			messages = append(messages, Message{
				Role:       "tool",
				ToolCallID: tc.Name,
				Content:    toolResult,
			})
		}

		if prCreated {
			break
		}
	}

	return nil
}

func (a *Agent) executeTool(ctx context.Context, tc ToolCall) (string, bool) {
	switch tc.Name {
	case "search_repo":
		query, _ := tc.Arguments["query"].(string)
		files, err := a.adapter.SearchRepo(ctx, query)
		if err != nil {
			return fmt.Sprintf("Error searching repo: %v", err), false
		}
		if len(files) == 0 {
			return "No files found matching query.", false
		}
		return fmt.Sprintf("Found files:\n%s", strings.Join(files, "\n")), false

	case "read_file":
		path, _ := tc.Arguments["path"].(string)
		content, err := a.adapter.ReadFile(ctx, path)
		if err != nil {
			return fmt.Sprintf("Error reading file %s: %v", path, err), false
		}
		return content, false

	case "patch_file":
		path, _ := tc.Arguments["path"].(string)
		content, _ := tc.Arguments["content"].(string)
		msg, _ := tc.Arguments["commit_message"].(string)
		if msg == "" {
			msg = fmt.Sprintf("Update %s based on latest patch notes", path)
		}
		err := a.adapter.UpdateFile(ctx, path, content, msg)
		if err != nil {
			return fmt.Sprintf("Error patching file %s: %v", path, err), false
		}
		return fmt.Sprintf("Successfully staged updates for %s", path), false

	case "create_pull_request":
		title, _ := tc.Arguments["title"].(string)
		body, _ := tc.Arguments["body"].(string)
		if title == "" {
			title = "Documentation updates for recent patch"
		}
		url, err := a.adapter.CreatePullRequest(ctx, title, body, "")
		if err != nil {
			return fmt.Sprintf("Error creating pull request: %v", err), false
		}
		return fmt.Sprintf("Pull request created successfully: %s", url), true

	default:
		return fmt.Sprintf("Unknown tool: %s", tc.Name), false
	}
}

