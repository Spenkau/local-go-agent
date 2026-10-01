package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	http  *http.Client
	url   string
	model string
}

func NewClient(url, model string) *Client {
	return &Client{
		http:  &http.Client{Timeout: 2 * time.Minute},
		url:   url,
		model: model,
	}
}

func (c *Client) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
) (Message, error) {
	body, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Think:    false,
		Stream:   false,
	})
	if err != nil {
		return Message{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return Message{}, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Message{}, fmt.Errorf("ollama returned %s: %s", resp.Status, data)
	}

	var out chatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return Message{}, fmt.Errorf("decode response: %w", err)
	}

	return out.Message, nil
}
