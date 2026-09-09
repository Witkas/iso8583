// Package openai implements llm.Generator against the OpenAI Chat Completions
// API using only the standard library — no SDK dependency is added to the
// module. The API key is supplied explicitly or read from OPENAI_API_KEY; the
// model defaults to a cheap capable model and can be overridden with
// ISO8583_OPENAI_MODEL.
package openai

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
	endpoint     = "https://api.openai.com/v1/chat/completions"
	defaultModel = "gpt-4o-mini"
)

// Client is an llm.Generator backed by OpenAI.
type Client struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// New returns a Client. If apiKey is empty it falls back to OPENAI_API_KEY. The
// model is ISO8583_OPENAI_MODEL if set, otherwise a cheap capable default.
func New(apiKey string) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	model := os.Getenv("ISO8583_OPENAI_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Client{APIKey: apiKey, Model: model, HTTP: &http.Client{Timeout: 60 * time.Second}}
}

type request struct {
	Model          string         `json:"model"`
	Messages       []message      `json:"messages"`
	ResponseFormat responseFormat `json:"response_format"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Describe implements llm.Generator.
func (c *Client) Describe(ctx context.Context, prompt string, cat llm.Catalog) (llm.Draft, error) {
	if c.APIKey == "" {
		return llm.Draft{}, fmt.Errorf("openai: no API key (set OPENAI_API_KEY)")
	}
	body, err := json.Marshal(request{
		Model: c.Model,
		Messages: []message{
			{Role: "system", Content: llm.SystemPrompt(cat)},
			{Role: "user", Content: prompt},
		},
		// json_object mode is satisfied because the system prompt asks for JSON.
		ResponseFormat: responseFormat{Type: "json_object"},
	})
	if err != nil {
		return llm.Draft{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return llm.Draft{}, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return llm.Draft{}, fmt.Errorf("openai: request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		return llm.Draft{}, fmt.Errorf("openai: unreadable response (HTTP %d): %s", resp.StatusCode, truncate(raw))
	}
	if out.Error != nil {
		return llm.Draft{}, fmt.Errorf("openai: API error (%s): %s", out.Error.Type, out.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return llm.Draft{}, fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, truncate(raw))
	}
	if len(out.Choices) == 0 || out.Choices[0].Message.Content == "" {
		return llm.Draft{}, fmt.Errorf("openai: empty response")
	}
	return llm.ParseDraftJSON(out.Choices[0].Message.Content)
}

func truncate(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
