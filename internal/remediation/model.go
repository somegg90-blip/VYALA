package remediation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Model is the minimal LLM interface the agents depend on. Everything is
// mockable so the pipeline can be tested without a running model.
type Model interface {
	Complete(ctx context.Context, req CompletionRequest) (string, error)
}

type CompletionRequest struct {
	System      string
	User        string
	Temperature float64
}

// OllamaClient talks to a local Ollama server (default http://localhost:11434)
// using /api/chat with format:"json" for grammar-constrained structured output.
type OllamaClient struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
	}
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model     string          `json:"model"`
	Messages  []ollamaMessage `json:"messages"`
	Stream    bool            `json:"stream"`
	Format    string          `json:"format"`
	Options   map[string]any  `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error"`
}

func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	payload := ollamaChatRequest{
		Model: c.Model,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		Stream: false,
		Format: "json",
		Options: map[string]any{
			"temperature": req.Temperature,
			"num_ctx":     8192,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama at %s unreachable (is it running? `ollama serve`): %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, buf.String())
	}

	var out ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decoding ollama response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("ollama error: %s", out.Error)
	}
	return out.Message.Content, nil
}

// ExtractJSON salvages a JSON object from a model response that may be wrapped
// in markdown fences or surrounded by prose. Small models do this constantly.
func ExtractJSON(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON object found in model output")
	}
	return []byte(s[start : end+1]), nil
}
