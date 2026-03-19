package agent

import "context"

// LLMProvider is the interface for an LLM-backed agent provider.
// Each provider manages its own agentic tool-calling loop.
type LLMProvider interface {
	Ask(ctx context.Context, question string) (string, error)
	Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error)
	GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	Close()
}

// ProviderConfig selects and configures the LLM provider.
type ProviderConfig struct {
	Provider   string  // "gemini" (default) | "ollama"
	GeminiKey  string
	OllamaURL  string  // default: http://localhost:11434
	OllamaModel string // default: llama3.2
}
