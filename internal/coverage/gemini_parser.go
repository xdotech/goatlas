package coverage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiParser uses the Gemini API to extract structured feature components from spec text.
type GeminiParser struct {
	client *genai.Client
	model  string
}

// NewGeminiParser creates a new GeminiParser using the provided API key.
func NewGeminiParser(ctx context.Context, apiKey string) (*GeminiParser, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &GeminiParser{client: client, model: "gemini-2.0-flash"}, nil
}

// Close releases resources held by the Gemini client.
func (p *GeminiParser) Close() {
	p.client.Close()
}

type featureJSON struct {
	Backend  []componentJSON `json:"backend"`
	Frontend []componentJSON `json:"frontend"`
}

type componentJSON struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

// ExtractFeatureComponents calls Gemini to parse a FeatureSection into a Feature with typed components.
func (p *GeminiParser) ExtractFeatureComponents(ctx context.Context, section FeatureSection) (*Feature, error) {
	model := p.client.GenerativeModel(p.model)
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(`Extract backend and frontend components from this feature specification.

Feature: %s

Spec text:
%s

Return JSON with this exact structure:
{
  "backend": [
    {"type": "api_endpoint", "identifier": "POST /path"},
    {"type": "service_method", "identifier": "ServiceName.MethodName"}
  ],
  "frontend": [
    {"type": "ui_screen", "identifier": "ScreenComponentName"},
    {"type": "api_call", "identifier": "POST /path"}
  ]
}

Types: api_endpoint, service_method, ui_screen, api_call
Only include components explicitly mentioned in the spec.`, section.Name, section.RawText)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	var rawJSON string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			rawJSON = string(txt)
			break
		}
	}

	// Clean up JSON (remove markdown code blocks if present)
	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var fj featureJSON
	if err := json.Unmarshal([]byte(rawJSON), &fj); err != nil {
		// Fallback: return feature with no components
		return &Feature{Name: section.Name, Description: section.RawText}, nil
	}

	feature := &Feature{
		Name:        section.Name,
		Description: section.RawText,
	}
	for _, c := range fj.Backend {
		feature.Backend = append(feature.Backend, Component{Type: c.Type, Identifier: c.Identifier})
	}
	for _, c := range fj.Frontend {
		feature.Frontend = append(feature.Frontend, Component{Type: c.Type, Identifier: c.Identifier})
	}
	return feature, nil
}

// ParseWithRegex extracts components using regex patterns (no AI, --no-ai flag).
func ParseWithRegex(section FeatureSection) *Feature {
	feature := &Feature{Name: section.Name, Description: section.RawText}

	lines := strings.Split(section.RawText, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
			if strings.Contains(line, method+" /") {
				idx := strings.Index(line, method+" /")
				identifier := extractURLFromLine(line[idx:])
				feature.Backend = append(feature.Backend, Component{
					Type:       "api_endpoint",
					Identifier: identifier,
				})
			}
		}
	}
	return feature
}

func extractURLFromLine(s string) string {
	parts := strings.Fields(s)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return s
}
