package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"

	"local-agent/internal/agent"
	"local-agent/internal/ollama"
	"local-agent/internal/tools"
)

const (
	modelName  = "qwen3:4b"
	ollamaURL  = "http://127.0.0.1:11434/api/chat"
	sandboxDir = "./sandbox"
	maxSteps   = 8
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	registry, err := tools.NewDefaultRegistry(sandboxDir)
	if err != nil {
		log.Fatalf("init tools: %v", err)
	}

	a := agent.New(ollama.NewClient(ollamaURL, modelName), registry, maxSteps)

	messages := []ollama.Message{{
		Role: "system",
		Content: "Ты полезный ассистент, отвечаешь на русском. " +
			"Для вопроса о текущем времени обязательно вызывай инструмент get_current_time. " +
			"Не придумывай текущее время самостоятельно.",
	}}

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nТы: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nЗавершение.")
				return
			}
			fmt.Printf("Ошибка ввода: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if input == "exit" {
			return
		}

		prev := len(messages)
		messages = append(messages, ollama.Message{Role: "user", Content: input})

		answer, updated, err := a.Run(ctx, messages)
		if err != nil {
			fmt.Printf("Ошибка агента: %v\n", err)
			messages = messages[:prev]
			continue
		}

		messages = updated
		fmt.Printf("Агент: %s\n", answer)
	}
}
