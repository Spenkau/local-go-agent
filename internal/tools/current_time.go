package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"local-agent/internal/ollama"
)

type CurrentTime struct{}

func (CurrentTime) Name() string { return "get_current_time" }

func (t CurrentTime) Definition() ollama.ToolDefinition {
	return ollama.ToolDefinition{
		Type: "function",
		Function: ollama.ToolDefinitionBody{
			Name:        t.Name(),
			Description: "Возвращает текущие дату и время. Поддерживается только Europe/Minsk.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"timezone": map[string]any{
						"type":        "string",
						"description": "IANA timezone, например Europe/Minsk",
						"enum":        []string{"Europe/Minsk"},
					},
				},
			},
		},
	}
}

func (t CurrentTime) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("decode arguments: %w", err)
	}

	if args.Timezone == "" {
		args.Timezone = "Europe/Minsk"
	}
	if args.Timezone != "Europe/Minsk" {
		return "", fmt.Errorf("timezone %q is not allowed", args.Timezone)
	}

	loc, err := time.LoadLocation(args.Timezone)
	if err != nil {
		return "", fmt.Errorf("load timezone: %w", err)
	}

	return time.Now().In(loc).Format("2006-01-02 15:04:05 MST"), nil
}
