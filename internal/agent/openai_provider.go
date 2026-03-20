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

// openaiProvider implements LLMProvider using any OpenAI-compatible API.
// Works with vLLM, LiteLLM, text-generation-inference, LocalAI, etc.
type openaiProvider struct {
	cfg             AgentConfig
	baseURL         string // e.g. "http://10.1.1.246:8001/v1"
	apiKey          string
	model           string
	bridge          *ToolBridge
	systemPrompt    string
	client          *http.Client
	tools           []openaiTool // cached at construction
	disableThinking bool         // disable reasoning/thinking mode (e.g. Qwen3)
}

func newOpenAIProvider(cfg AgentConfig, provCfg ProviderConfig, bridge *ToolBridge, systemPrompt string) (*openaiProvider, error) {
	url := provCfg.OpenAIBaseURL
	if url == "" {
		url = "http://localhost:8001/v1"
	}
	// Strip trailing slash for clean URL joining
	url = strings.TrimRight(url, "/")

	apiKey := provCfg.OpenAIAPIKey
	if apiKey == "" {
		apiKey = "ignored"
	}
	model := provCfg.OpenAIModel
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	return &openaiProvider{
		cfg:             cfg,
		baseURL:         url,
		apiKey:          apiKey,
		model:           model,
		bridge:          bridge,
		systemPrompt:    systemPrompt,
		client:          &http.Client{},
		tools:           buildOpenAITools(),
		disableThinking: provCfg.OpenAIDisableThinking,
	}, nil
}

func (p *openaiProvider) Close() {}

func (p *openaiProvider) Ask(ctx context.Context, question string) (string, error) {
	messages := []openaiMessage{
		{Role: "system", Content: p.systemPrompt},
		{Role: "user", Content: question},
	}
	return p.runLoop(ctx, messages)
}

func (p *openaiProvider) Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error) {
	messages := []openaiMessage{{Role: "system", Content: p.systemPrompt}}
	for _, h := range history {
		role := h.Role
		if role == "model" {
			role = "assistant"
		}
		messages = append(messages, openaiMessage{Role: role, Content: h.Content})
	}
	messages = append(messages, openaiMessage{Role: "user", Content: message})

	result, err := p.runLoop(ctx, messages)
	if err != nil {
		return "", history, err
	}
	history = append(history, ConversationMessage{Role: "user", Content: message})
	history = append(history, ConversationMessage{Role: "model", Content: result})
	return result, history, nil
}

func (p *openaiProvider) GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	messages := []openaiMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	reqBody := openaiChatRequest{
		Model:    p.model,
		Messages: messages,
	}
	if p.disableThinking {
		reqBody.ChatTemplateKwargs = &chatTemplateKwargs{EnableThinking: false}
	}
	return p.chatOnce(ctx, reqBody)
}

// runLoop drives the OpenAI tool-calling loop.
func (p *openaiProvider) runLoop(ctx context.Context, messages []openaiMessage) (string, error) {
	for i := 0; i < p.cfg.MaxIterations; i++ {
		reqBody := openaiChatRequest{
			Model:    p.model,
			Messages: messages,
			Tools:    p.tools,
		}
		if p.disableThinking {
			reqBody.ChatTemplateKwargs = &chatTemplateKwargs{EnableThinking: false}
		}

		resp, err := p.sendChat(ctx, reqBody)
		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("openai: empty choices in response")
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return strings.TrimSpace(msg.Content), nil
		}

		// Append assistant message with tool calls
		messages = append(messages, openaiMessage{
			Role:      "assistant",
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		// Execute each tool call and append results
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{}
			}
			result, execErr := p.bridge.Execute(ctx, tc.Function.Name, args)
			if execErr != nil {
				result = fmt.Sprintf("Error executing %s: %v", tc.Function.Name, execErr)
			}
			messages = append(messages, openaiMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "Maximum iterations reached", nil
}

func (p *openaiProvider) chatOnce(ctx context.Context, req openaiChatRequest) (string, error) {
	resp, err := p.sendChat(ctx, req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("openai: empty choices")
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

func (p *openaiProvider) sendChat(ctx context.Context, req openaiChatRequest) (*openaiChatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build openai request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai chat: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("openai chat: status %d: %s", httpResp.StatusCode, errBody)
	}

	var resp openaiChatResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	return &resp, nil
}

// --- OpenAI API types ---

type openaiMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	ToolCalls  []openaiToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string                   `json:"id"`
	Type     string                   `json:"type"` // "function"
	Function openaiToolCallFunction   `json:"function"`
}

type openaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type openaiChatRequest struct {
	Model              string                `json:"model"`
	Messages           []openaiMessage       `json:"messages"`
	Tools              []openaiTool          `json:"tools,omitempty"`
	ChatTemplateKwargs *chatTemplateKwargs   `json:"chat_template_kwargs,omitempty"`
}

// chatTemplateKwargs passes template options to vLLM (e.g. disable Qwen3 thinking).
type chatTemplateKwargs struct {
	EnableThinking bool `json:"enable_thinking"`
}

type openaiChatResponse struct {
	Choices []openaiChoice `json:"choices"`
}

type openaiChoice struct {
	Message openaiMessage `json:"message"`
}

type openaiTool struct {
	Type     string             `json:"type"` // "function"
	Function openaiToolFunction `json:"function"`
}

type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// buildOpenAITools converts generic tool schemas to OpenAI format.
func buildOpenAITools() []openaiTool {
	schemas := toolSchemas()
	tools := make([]openaiTool, 0, len(schemas))
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
		tools = append(tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
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
