package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	modelName = "qwen3:4b"
	ollamaURL = "http://127.0.0.1:11434/api/chat"
)

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
}

type ToolCall struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolDefinitionFunction `json:"function"`
}

type ToolDefinitionFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Think    bool             `json:"think"`
	Stream   bool             `json:"stream"`
}

type ChatResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

type TimeArguments struct {
	Timezone string `json:"timezone"`
}

type FileArguments struct {
	Path string `json:"path"`
}

func main() {
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}

	messages := []Message{
		{
			Role: "system",
			Content: "Ты полезный ассистент, отвечаешь на русском. " +
				"Для вопроса о текущем времени обязательно вызывай инструмент get_current_time. " +
				"Не придумывай текущее время самостоятельно.",
		},
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nТы: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nЗавершение.")
				return
			}

			fmt.Printf("Ошибка чтения ввода: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		if input == "exit" {
			fmt.Println("Завершение.")
			return
		}

		messages = append(messages, Message{
			Role:    "user",
			Content: input,
		})

		answer, updatedMessages, err := runAgent(client, messages)
		if err != nil {
			fmt.Printf("Ошибка агента: %v\n", err)

			messages = messages[:len(messages)-1]
			continue
		}

		messages = updatedMessages

		fmt.Printf("Агент: %s\n", answer)
	}
}

func runAgent(
	client *http.Client,
	messages []Message,
) (string, []Message, error) {
	for step := 0; step < 3; step++ {
		response, err := askOllama(client, messages)
		if err != nil {
			return "", messages, err
		}

		messages = append(messages, response.Message)

		if len(response.Message.ToolCalls) == 0 {
			return response.Message.Content, messages, nil
		}

		for _, toolCall := range response.Message.ToolCalls {
			result, err := executeTool(toolCall)
			if err != nil {
				return "", messages, err
			}

			messages = append(messages, Message{
				Role:     "tool",
				ToolName: toolCall.Function.Name,
				Content:  result,
			})
		}
	}

	return "", messages, fmt.Errorf("agent exceeded maximum tool-call steps")
}

func askOllama(
	client *http.Client,
	messages []Message,
) (ChatResponse, error) {
	requestBody := ChatRequest{
		Model:    modelName,
		Messages: messages,
		Tools:    availableTools(),
		Think:    false,
		Stream:   false,
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ollamaURL,
		bytes.NewReader(body),
	)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call Ollama: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return ChatResponse{}, fmt.Errorf(
			"ollama returned %s: %s",
			response.Status,
			string(responseBody),
		)
	}

	var chatResponse ChatResponse

	if err := json.Unmarshal(responseBody, &chatResponse); err != nil {
		return ChatResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return chatResponse, nil
}

func availableTools() ([]ToolDefinition, error) {
	toolsFile, err := os.ReadFile("tools.json")

	if err != nil {
		return nil, fmt.Errorf("failed to read tools.json: %w", err)
	}

	var toolDefinitions []ToolDefinition
	err = json.Unmarshal(toolsFile, &toolDefinitions)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools.json: %w", err)
	}

	return toolDefinitions, nil
}

type ToolFunc func(rawArguments json.RawMessage) (string, error)

var ToolRegistry = map[string]ToolFunc{
	"get_current_time":  getCurrentTime,
	"read_project_file": readProjectFile,
}

func executeTool(toolCall ToolCall) (string, error) {
	result, err := ToolRegistry[toolCall.Function.Name](toolCall.Function.Arguments)

	if err != nil {
		return "", fmt.Errorf(
			"tool %q is not allowed",
			toolCall.Function.Name,
		)
	}

	return result, nil
}

func getCurrentTime(rawArguments json.RawMessage) (string, error) {
	var arguments TimeArguments

	if err := json.Unmarshal(rawArguments, &arguments); err != nil {
		return "", fmt.Errorf("decode get_current_time arguments: %w", err)
	}

	if arguments.Timezone == "" {
		arguments.Timezone = "Europe/Minsk"
	}

	if arguments.Timezone != "Europe/Minsk" {
		return "", fmt.Errorf(
			"timezone %q is not allowed",
			arguments.Timezone,
		)
	}

	location, err := time.LoadLocation(arguments.Timezone)
	if err != nil {
		return "", fmt.Errorf("load timezone: %w", err)
	}

	return time.Now().
		In(location).
		Format("2006-01-02 15:04:05 MST"), nil
}

func readProjectFile(rawArguments json.RawMessage) (string, error) {
	var arguments FileArguments

	if err := json.Unmarshal(rawArguments, &arguments); err != nil {
		return "", fmt.Errorf("decode read_project_file arguments: %w", err)
	}

	if !fs.ValidPath(arguments.Path) {
		return "", fmt.Errorf("invalid relative file path")
	}

	securedDir := os.DirFS("./sandbox")

	const maxFileSize = 64 * 1024

	file, err := securedDir.Open(arguments.Path)
	if err != nil {
		return "", fmt.Errorf("file is unavailable")
	}
	defer file.Close()

	limitedReader := io.LimitReader(file, maxFileSize)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("read project file: %w", err)
	}

	return string(data), nil
}
