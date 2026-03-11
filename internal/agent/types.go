package agent

// AgentConfig configures the Gemini agent.
type AgentConfig struct {
	MaxIterations int
	Model         string
	Temperature   float32
	RepoName      string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() AgentConfig {
	return AgentConfig{
		MaxIterations: 20,
		Model:         "gemini-2.0-flash",
		Temperature:   0.1,
		RepoName:      "the repository",
	}
}

// ConversationMessage is a single turn in a conversation.
type ConversationMessage struct {
	Role    string // "user" | "model"
	Content string
}

// ToolCall records an agent tool invocation.
type ToolCall struct {
	Name   string
	Args   map[string]interface{}
	Result string
}
