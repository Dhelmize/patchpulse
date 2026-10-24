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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"patchpulse/internal/config"
)

type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type LLMProvider interface {
	Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*Message, error)
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func GetDefaultTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "search_repo",
			Description: "Search for files within the repository/wiki matching a query or filename",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search term or file path fragment",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "read_file",
			Description: "Read the entire text content of a specified file or wiki page",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to file or title of wiki page",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "patch_file",
			Description: "Update the content of a file or wiki page with new documentation or stats",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to file or title of wiki page",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Complete new content for the file",
					},
					"commit_message": map[string]interface{}{
						"type":        "string",
						"description": "Concise summary of modifications made",
					},
				},
				"required": []string{"path", "content", "commit_message"},
			},
		},
		{
			Name:        "create_pull_request",
			Description: "Finalize all staged changes into a pull request or publish wiki revisions",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Title of the pull request or update summary",
					},
					"body": map[string]interface{}{
						"type":        "string",
						"description": "Markdown body detailing what was updated",
					},
				},
				"required": []string{"title", "body"},
			},
		},
	}
}

// OpenAI / OpenAI-Compatible Provider

type OpenAIProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func NewOpenAIProvider(apiKey, model, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "gpt-4"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *OpenAIProvider) Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*Message, error) {
	type openAITool struct {
		Type     string         `json:"type"`
		Function ToolDefinition `json:"function"`
	}

	var formattedTools []openAITool
	for _, t := range tools {
		formattedTools = append(formattedTools, openAITool{
			Type:     "function",
			Function: t,
		})
	}

	type openAIMsg struct {
		Role       string      `json:"role"`
		Content    interface{} `json:"content"`
		ToolCalls  interface{} `json:"tool_calls,omitempty"`
		ToolCallID string      `json:"tool_call_id,omitempty"`
	}

	var formattedMsgs []openAIMsg
	for _, m := range messages {
		msg := openAIMsg{
			Role:    m.Role,
			Content: m.Content,
		}
		if len(m.ToolCalls) > 0 {
			var tcList []map[string]interface{}
			for _, tc := range m.ToolCalls {
				argBytes, _ := json.Marshal(tc.Arguments)
				tcList = append(tcList, map[string]interface{}{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      tc.Name,
						"arguments": string(argBytes),
					},
				})
			}
			msg.ToolCalls = tcList
		}
		if m.ToolCallID != "" {
			msg.ToolCallID = m.ToolCallID
		}
		formattedMsgs = append(formattedMsgs, msg)
	}

	reqBody := map[string]interface{}{
		"model":    p.model,
		"messages": formattedMsgs,
	}
	if len(formattedTools) > 0 {
		reqBody["tools"] = formattedTools
		reqBody["tool_choice"] = "auto"
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/chat/completions", p.baseURL), bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, err
	}

	if len(res.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned by LLM")
	}

	choice := res.Choices[0].Message
	resultMsg := &Message{
		Role:    choice.Role,
		Content: choice.Content,
	}

	for _, tc := range choice.ToolCalls {
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		resultMsg.ToolCalls = append(resultMsg.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	return resultMsg, nil
}

// Google Gemini Provider (REST v1beta with Function Calling)

type GeminiProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewGeminiProvider(apiKey, model string) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GeminiProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

func (p *GeminiProvider) Generate(ctx context.Context, messages []Message, tools []ToolDefinition) (*Message, error) {
	// Build contents array for Gemini
	type geminiPart struct {
		Text             string      `json:"text,omitempty"`
		FunctionCall     interface{} `json:"functionCall,omitempty"`
		FunctionResponse interface{} `json:"functionResponse,omitempty"`
	}

	type geminiContent struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	}

	var systemInstruction *geminiContent
	var contents []geminiContent

	for _, m := range messages {
		if m.Role == "system" {
			systemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}

		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}

		var parts []geminiPart
		if m.Content != "" {
			parts = append(parts, geminiPart{Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			parts = append(parts, geminiPart{
				FunctionCall: map[string]interface{}{
					"name": tc.Name,
					"args": tc.Arguments,
				},
			})
		}
		if m.Role == "tool" {
			role = "user"
			parts = append(parts, geminiPart{
				FunctionResponse: map[string]interface{}{
					"name": m.ToolCallID,
					"response": map[string]interface{}{
						"content": m.Content,
					},
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, geminiContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	// Function declarations for Gemini tools
	type funcDecl struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	var funcDecls []funcDecl
	for _, t := range tools {
		funcDecls = append(funcDecls, funcDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}

	reqBody := map[string]interface{}{
		"contents": contents,
	}
	if systemInstruction != nil {
		reqBody["systemInstruction"] = systemInstruction
	}
	if len(funcDecls) > 0 {
		reqBody["tools"] = []map[string]interface{}{
			{"functionDeclarations": funcDecls},
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, err
	}

	if len(res.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned by Gemini")
	}

	candidate := res.Candidates[0].Content
	resultMsg := &Message{Role: "assistant"}

	var textParts []string
	for _, part := range candidate.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			resultMsg.ToolCalls = append(resultMsg.ToolCalls, ToolCall{
				ID:        part.FunctionCall.Name,
				Name:      part.FunctionCall.Name,
				Arguments: part.FunctionCall.Args,
			})
		}
	}
	resultMsg.Content = strings.Join(textParts, "\n")

	return resultMsg, nil
}

func NewLLMProvider(cfg *config.Config) (LLMProvider, error) {
	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}

	if strings.ToLower(cfg.LLMProvider) == "gemini" {
		return NewGeminiProvider(cfg.LLMAPIKey, cfg.LLMModel), nil
	}

	return NewOpenAIProvider(cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMBaseURL), nil
}

