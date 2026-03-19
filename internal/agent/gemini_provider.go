package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// geminiProvider implements LLMProvider using the Gemini API.
type geminiProvider struct {
	cfg          AgentConfig
	client       *genai.Client
	bridge       *ToolBridge
	systemPrompt string
}

func newGeminiProvider(ctx context.Context, cfg AgentConfig, apiKey string, bridge *ToolBridge, systemPrompt string) (*geminiProvider, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &geminiProvider{cfg: cfg, client: client, bridge: bridge, systemPrompt: systemPrompt}, nil
}

func (p *geminiProvider) Close() {
	p.client.Close()
}

func (p *geminiProvider) Ask(ctx context.Context, question string) (string, error) {
	session := p.newSession()
	return p.runLoop(ctx, session, question)
}

func (p *geminiProvider) Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error) {
	session := p.newSession()
	for _, msg := range history {
		if msg.Role == "user" || msg.Role == "model" {
			session.History = append(session.History, &genai.Content{
				Role:  msg.Role,
				Parts: []genai.Part{genai.Text(msg.Content)},
			})
		}
	}

	result, err := p.runLoop(ctx, session, message)
	if err != nil {
		return "", history, err
	}
	history = append(history, ConversationMessage{Role: "user", Content: message})
	history = append(history, ConversationMessage{Role: "model", Content: result})
	return result, history, nil
}

func (p *geminiProvider) GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	temp := p.cfg.Temperature
	model := p.client.GenerativeModel(p.cfg.Model)
	model.Temperature = &temp
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(systemPrompt)},
		Role:  "system",
	}

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("no response from model")
	}

	var parts []string
	for _, part := range resp.Candidates[0].Content.Parts {
		if t, ok := part.(genai.Text); ok {
			parts = append(parts, string(t))
		}
	}
	return strings.Join(parts, "\n"), nil
}

func (p *geminiProvider) newSession() *genai.ChatSession {
	temp := p.cfg.Temperature
	model := p.client.GenerativeModel(p.cfg.Model)
	model.Temperature = &temp
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{genai.Text(p.systemPrompt)},
		Role:  "system",
	}
	model.Tools = []*genai.Tool{{FunctionDeclarations: toolDeclarations()}}
	return model.StartChat()
}

func (p *geminiProvider) runLoop(ctx context.Context, session *genai.ChatSession, input string) (string, error) {
	parts := []genai.Part{genai.Text(input)}

	for i := 0; i < p.cfg.MaxIterations; i++ {
		resp, err := session.SendMessage(ctx, parts...)
		if err != nil {
			return "", fmt.Errorf("gemini send: %w", err)
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			return "No response from model", nil
		}
		content := resp.Candidates[0].Content

		var funcCalls []genai.FunctionCall
		var textParts []string
		for _, part := range content.Parts {
			switch v := part.(type) {
			case genai.FunctionCall:
				funcCalls = append(funcCalls, v)
			case genai.Text:
				textParts = append(textParts, string(v))
			}
		}

		if len(funcCalls) == 0 {
			return strings.Join(textParts, "\n"), nil
		}

		responseParts := make([]genai.Part, 0, len(funcCalls))
		for _, fc := range funcCalls {
			result, execErr := p.bridge.Execute(ctx, fc.Name, fc.Args)
			if execErr != nil {
				result = fmt.Sprintf("Error executing %s: %v", fc.Name, execErr)
			}
			responseParts = append(responseParts, genai.FunctionResponse{
				Name:     fc.Name,
				Response: map[string]any{"result": result},
			})
		}
		parts = responseParts
	}

	return "Maximum iterations reached", nil
}
