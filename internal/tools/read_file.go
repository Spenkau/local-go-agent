package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"

	"local-agent/internal/ollama"
)

type ReadProjectFile struct {
	FS      fs.FS
	MaxSize int64
}

func (ReadProjectFile) Name() string { return "read_project_file" }

func (t ReadProjectFile) Definition() ollama.ToolDefinition {
	return ollama.ToolDefinition{
		Type: "function",
		Function: ollama.ToolDefinitionBody{
			Name:        t.Name(),
			Description: "Читает текстовый файл из песочницы проекта по относительному пути.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Относительный путь, например notes/todo.txt",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

func (t ReadProjectFile) Execute(_ context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("decode arguments: %w", err)
	}

	if !fs.ValidPath(args.Path) {
		return "", fmt.Errorf("invalid relative file path")
	}

	f, err := t.FS.Open(args.Path)
	if err != nil {
		return "", fmt.Errorf("file is unavailable")
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, t.MaxSize))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return string(data), nil
}
