// Package claude implements llm.Generator against the Anthropic Messages API
// using only the standard library — no SDK dependency is added to the module.
// The API key is supplied explicitly or read from ANTHROPIC_API_KEY; the model
// defaults to the cheapest capable Claude model and can be overridden with
// ISO8583_CLAUDE_MODEL.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Witkas/iso8583/llm"
)

const (
	endpoint     = "https://api.anthropic.com/v1/messages"
	apiVersion   = "2023-06-01"
	defaultModel = "claude-haiku-4-5"
)

// Client is an llm.Generator backed by Claude.
type Client struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// New returns a Client. If apiKey is empty it falls back to ANTHROPIC_API_KEY.
// The model is ISO8583_CLAUDE_MODEL if set, otherwise the cheapest capable
// default.
func New(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	model := os.Getenv("ISO8583_CLAUDE_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Client{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Describe implements llm.Generator.
func (c *Client) Describe(ctx context.Context, prompt string, cat llm.Catalog) (llm.Draft, error) {
	if c.APIKey == "" {
		return llm.Draft{}, fmt.Errorf("claude: no API key (set ANTHROPIC_API_KEY)")
	}
	body, err := json.Marshal(request{
		Model:     c.Model,
		MaxTokens: 1024,
		System:    llm.SystemPrompt(cat),
		Messages:  []message{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return llm.Draft{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return llm.Draft{}, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return llm.Draft{}, fmt.Errorf("claude: request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		return llm.Draft{}, fmt.Errorf("claude: unreadable response (HTTP %d): %s", resp.StatusCode, truncate(raw))
	}
	if out.Error != nil {
		return llm.Draft{}, fmt.Errorf("claude: API error (%s): %s", out.Error.Type, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return llm.Draft{}, fmt.Errorf("claude: HTTP %d: %s", resp.StatusCode, truncate(raw))
	}

	text := firstText(out)
	if text == "" {
		return llm.Draft{}, fmt.Errorf("claude: empty response")
	}
	return llm.ParseDraftJSON(text)
}

func firstText(r response) string {
	for _, b := range r.Content {
		if b.Type == "text" && b.Text != "" {
			return b.Text
		}
	}
	return ""
}

func truncate(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
