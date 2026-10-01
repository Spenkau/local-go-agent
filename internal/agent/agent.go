package agent

import (
	"context"
	"fmt"
	"log"

	"local-agent/internal/ollama"
	"local-agent/internal/tools"
)

type Agent struct {
	llm      *ollama.Client
	registry *tools.Registry
	maxSteps int
}

func New(llm *ollama.Client, registry *tools.Registry, maxSteps int) *Agent {
	return &Agent{llm: llm, registry: registry, maxSteps: maxSteps}
}

func (a *Agent) Run(
	ctx context.Context,
	messages []ollama.Message,
) (string, []ollama.Message, error) {
	defs := a.registry.Definitions()

	for step := 0; step < a.maxSteps; step++ {
		reply, err := a.llm.Chat(ctx, messages, defs)
		if err != nil {
			return "", messages, err
		}

		messages = append(messages, reply)

		if len(reply.ToolCalls) == 0 {
			return reply.Content, messages, nil
		}

		for _, call := range reply.ToolCalls {
			name := call.Function.Name
			log.Printf("tool call: %s %s", name, call.Function.Arguments)

			result, err := a.registry.Execute(ctx, name, call.Function.Arguments)
			if err != nil {
				log.Printf("tool error: %s: %v", name, err)
				result = "error: " + err.Error()
			}

			messages = append(messages, ollama.Message{
				Role:     "tool",
				ToolName: name,
				Content:  result,
			})
		}
	}

	return "", messages, fmt.Errorf("agent exceeded %d steps", a.maxSteps)
}
