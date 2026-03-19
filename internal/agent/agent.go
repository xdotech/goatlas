package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Agent is the Gemini-powered code intelligence agent.
type Agent struct {
	cfg          AgentConfig
	client       *genai.Client
	bridge       *ToolBridge
	systemPrompt string
}

// NewAgent creates a new Gemini agent.
func NewAgent(ctx context.Context, cfg AgentConfig, apiKey string, bridge *ToolBridge, systemPrompt string) (*Agent, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &Agent{
		cfg:          cfg,
		client:       client,
		bridge:       bridge,
		systemPrompt: systemPrompt,
	}, nil
}

// Close releases the underlying client resources.
func (a *Agent) Close() {
	a.client.Close()
}

// Ask answers a single question using the agentic tool-calling loop.
func (a *Agent) Ask(ctx context.Context, question string) (string, error) {
	session := a.newSession()
	return a.runLoop(ctx, session, question)
}

// Chat runs one turn in a multi-turn conversation and returns the updated history.
func (a *Agent) Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error) {
	session := a.newSession()
	// Restore prior conversation history into the chat session.
	for _, msg := range history {
		if msg.Role == "user" || msg.Role == "model" {
			session.History = append(session.History, &genai.Content{
				Role:  msg.Role,
				Parts: []genai.Part{genai.Text(msg.Content)},
			})
		}
	}

	result, err := a.runLoop(ctx, session, message)
	if err != nil {
		return "", history, err
	}

	history = append(history, ConversationMessage{Role: "user", Content: message})
	history = append(history, ConversationMessage{Role: "model", Content: result})
	return result, history, nil
}

// newSession creates a configured chat session.
func (a *Agent) newSession() *genai.ChatSession {
	temp := a.cfg.Temperature
	model := a.client.GenerativeModel(a.cfg.Model)
	model.Temperature = &temp
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(a.systemPrompt)},
		Role:  "system",
	}
	model.Tools = a.buildToolDefinitions()
	return model.StartChat()
}

// runLoop drives the function-calling loop until the model returns a text response.
func (a *Agent) runLoop(ctx context.Context, session *genai.ChatSession, input string) (string, error) {
	parts := []genai.Part{genai.Text(input)}

	for i := 0; i < a.cfg.MaxIterations; i++ {
		resp, err := session.SendMessage(ctx, parts...)
		if err != nil {
			return "", fmt.Errorf("gemini send: %w", err)
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			return "No response from model", nil
		}
		content := resp.Candidates[0].Content

		// Separate function calls from text parts.
		var funcCalls []genai.FunctionCall
		var textParts []string
		for _, p := range content.Parts {
			switch v := p.(type) {
			case genai.FunctionCall:
				funcCalls = append(funcCalls, v)
			case genai.Text:
				textParts = append(textParts, string(v))
			}
		}

		// No function calls means the model has finished.
		if len(funcCalls) == 0 {
			return strings.Join(textParts, "\n"), nil
		}

		// Execute all function calls and collect FunctionResponse parts.
		responseParts := make([]genai.Part, 0, len(funcCalls))
		for _, fc := range funcCalls {
			result, execErr := a.bridge.Execute(ctx, fc.Name, fc.Args)
			if execErr != nil {
				result = fmt.Sprintf("Error executing %s: %v", fc.Name, execErr)
			}
			responseParts = append(responseParts, genai.FunctionResponse{
				Name:     fc.Name,
				Response: map[string]any{"result": result},
			})
		}

		// Feed all function results back in a single message.
		parts = responseParts
	}

	return "Maximum iterations reached", nil
}

// buildToolDefinitions returns the Gemini tool schema for all MCP tools.
func (a *Agent) buildToolDefinitions() []*genai.Tool {
	return []*genai.Tool{{FunctionDeclarations: toolDeclarations()}}
}

// GenerateDoc makes a one-shot Gemini call (no tool loop) with a custom system
// prompt. Used for generating documentation, SKILL.md, and wiki content.
func (a *Agent) GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	temp := a.cfg.Temperature
	model := a.client.GenerativeModel(a.cfg.Model)
	model.Temperature = &temp
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
		Role:  "system",
	}
	// No tools — pure text generation

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no response from model")
	}

	var parts []string
	for _, p := range resp.Candidates[0].Content.Parts {
		if t, ok := p.(genai.Text); ok {
			parts = append(parts, string(t))
		}
	}
	return strings.Join(parts, "\n"), nil
}

