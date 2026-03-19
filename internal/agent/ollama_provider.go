package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOllamaURL = "http://localhost:11434"
const defaultOllamaModel = "llama3.2"

// ollamaProvider implements LLMProvider using the Ollama API.
// Uses /api/chat with tool_calls support (Ollama ≥ 0.3).
type ollamaProvider struct {
	cfg          AgentConfig
	baseURL      string
	model        string
	bridge       *ToolBridge
	systemPrompt string
	client       *http.Client
	tools        []ollamaTool // cached at construction
}

func newOllamaProvider(cfg AgentConfig, provCfg ProviderConfig, bridge *ToolBridge, systemPrompt string) (*ollamaProvider, error) {
	url := provCfg.OllamaURL
	if url == "" {
		url = defaultOllamaURL
	}
	model := provCfg.OllamaModel
	if model == "" {
		model = defaultOllamaModel
	}
	return &ollamaProvider{
		cfg:          cfg,
		baseURL:      url,
		model:        model,
		bridge:       bridge,
		systemPrompt: systemPrompt,
		client:       &http.Client{},
		tools:        buildOllamaTools(),
	}, nil
}

func (p *ollamaProvider) Close() {}

func (p *ollamaProvider) Ask(ctx context.Context, question string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: p.systemPrompt},
		{Role: "user", Content: question},
	}
	return p.runLoop(ctx, messages)
}

func (p *ollamaProvider) Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error) {
	messages := []ollamaMessage{{Role: "system", Content: p.systemPrompt}}
	for _, h := range history {
		role := h.Role
		if role == "model" {
			role = "assistant"
		}
		messages = append(messages, ollamaMessage{Role: role, Content: h.Content})
	}
	messages = append(messages, ollamaMessage{Role: "user", Content: message})

	result, err := p.runLoop(ctx, messages)
	if err != nil {
		return "", history, err
	}
	history = append(history, ConversationMessage{Role: "user", Content: message})
	history = append(history, ConversationMessage{Role: "model", Content: result})
	return result, history, nil
}

func (p *ollamaProvider) GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []ollamaMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	reqBody := ollamaChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
	}
	return p.chatOnce(ctx, reqBody)
}

// runLoop drives the Ollama tool-calling loop.
func (p *ollamaProvider) runLoop(ctx context.Context, messages []ollamaMessage) (string, error) {
	for i := 0; i < p.cfg.MaxIterations; i++ {
		reqBody := ollamaChatRequest{
			Model:    p.model,
			Messages: messages,
			Tools:    p.tools,
			Stream:   false,
		}

		resp, err := p.sendChat(ctx, reqBody)
		if err != nil {
			return "", err
		}

		msg := resp.Message
		if len(msg.ToolCalls) == 0 {
			return strings.TrimSpace(msg.Content), nil
		}

		// Append assistant message with tool calls
		messages = append(messages, ollamaMessage{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		// Execute each tool call and append results
		for _, tc := range msg.ToolCalls {
			result, execErr := p.bridge.Execute(ctx, tc.Function.Name, tc.Function.Arguments)
			if execErr != nil {
				result = fmt.Sprintf("Error executing %s: %v", tc.Function.Name, execErr)
			}
			messages = append(messages, ollamaMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "Maximum iterations reached", nil
}

func (p *ollamaProvider) chatOnce(ctx context.Context, req ollamaChatRequest) (string, error) {
	resp, err := p.sendChat(ctx, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Message.Content), nil
}

func (p *ollamaProvider) sendChat(ctx context.Context, req ollamaChatRequest) (*ollamaChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama chat: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("ollama chat: status %d: %s", httpResp.StatusCode, errBody)
	}

	var resp ollamaChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}
	return &resp, nil
}

// --- Ollama API types ---

type ollamaMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type ollamaToolCall struct {
	ID       string                 `json:"id,omitempty"`
	Function ollamaToolCallFunction `json:"function"`
}

type ollamaToolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Tools    []ollamaTool    `json:"tools,omitempty"`
	Stream   bool            `json:"stream"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

type ollamaTool struct {
	Type     string              `json:"type"`
	Function ollamaToolFunction  `json:"function"`
}

type ollamaToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// buildOllamaTools converts generic tool schemas to Ollama format.
func buildOllamaTools() []ollamaTool {
	schemas := toolSchemas()
	tools := make([]ollamaTool, 0, len(schemas))
	for _, t := range schemas {
		props := map[string]interface{}{}
		for name, p := range t.Properties {
			props[name] = map[string]interface{}{
				"type":        p.Type,
				"description": p.Description,
			}
		}
		required := t.Required
		if required == nil {
			required = []string{}
		}
		tools = append(tools, ollamaTool{
			Type: "function",
			Function: ollamaToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		})
	}
	return tools
}
