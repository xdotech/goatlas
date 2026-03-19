package agent

import "context"

// Agent is the LLM-powered code intelligence agent.
// It delegates to a pluggable LLMProvider (Gemini, Ollama, etc.).
type Agent struct {
	provider LLMProvider
}

// NewAgent creates an Agent backed by the provider selected in provCfg.
func NewAgent(ctx context.Context, cfg AgentConfig, provCfg ProviderConfig, bridge *ToolBridge, systemPrompt string) (*Agent, error) {
	var provider LLMProvider
	var err error

	switch provCfg.Provider {
	case "ollama":
		provider, err = newOllamaProvider(cfg, provCfg, bridge, systemPrompt)
	default:
		provider, err = newGeminiProvider(ctx, cfg, provCfg.GeminiKey, bridge, systemPrompt)
	}
	if err != nil {
		return nil, err
	}
	return &Agent{provider: provider}, nil
}

// Close releases the underlying provider resources.
func (a *Agent) Close() {
	a.provider.Close()
}

// Ask answers a single question using the agentic tool-calling loop.
func (a *Agent) Ask(ctx context.Context, question string) (string, error) {
	return a.provider.Ask(ctx, question)
}

// Chat runs one turn in a multi-turn conversation and returns the updated history.
func (a *Agent) Chat(ctx context.Context, history []ConversationMessage, message string) (string, []ConversationMessage, error) {
	return a.provider.Chat(ctx, history, message)
}

// GenerateDoc makes a one-shot LLM call (no tool loop) with a custom system prompt.
func (a *Agent) GenerateDoc(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return a.provider.GenerateDoc(ctx, systemPrompt, userPrompt)
}
