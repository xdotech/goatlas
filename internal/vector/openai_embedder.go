package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openaiEmbedder calls an OpenAI-compatible /v1/embeddings endpoint.
// Works with vLLM, LiteLLM, text-generation-inference, LocalAI, etc.
type openaiEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

func newOpenAIEmbedder(cfg EmbedConfig) (*openaiEmbedder, error) {
	url := cfg.OpenAIBaseURL
	if url == "" {
		url = "http://localhost:8001/v1"
	}
	url = strings.TrimRight(url, "/")

	apiKey := cfg.OpenAIAPIKey
	if apiKey == "" {
		apiKey = "ignored"
	}
	model := cfg.OpenAIModel
	if model == "" {
		model = "text-embedding-ada-002"
	}
	return &openaiEmbedder{baseURL: url, apiKey: apiKey, model: model, client: &http.Client{}}, nil
}

func (e *openaiEmbedder) Close() {}

func (e *openaiEmbedder) EmbedOne(ctx context.Context, text string) ([]float32, error) {
	vecs, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (e *openaiEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(openaiEmbedRequest{
		Model: e.model,
		Input: texts,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal openai embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build openai embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai embed: status %d: %s", resp.StatusCode, errBody)
	}

	var result openaiEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode openai embed response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("openai returned empty embeddings")
	}

	// Sort by index to ensure correct order
	vecs := make([][]float32, len(result.Data))
	for _, d := range result.Data {
		if d.Index < len(vecs) {
			vecs[d.Index] = d.Embedding
		}
	}
	return vecs, nil
}

type openaiEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type openaiEmbedResponse struct {
	Data []openaiEmbedData `json:"data"`
}

type openaiEmbedData struct {
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}
