package agent

import (
	"context"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	Context  context.Context
	Model    *openai.LLM
	Tools    []llms.Tool
	Messages []llms.MessageContent
}

// Query sends a message to the agent and returns the response content
func (agent *Agent) Query(msg string) (*llms.ContentResponse, error) {
	agent.Messages = append(agent.Messages, llms.TextParts(llms.ChatMessageTypeHuman, msg))
	response, err := agent.Model.GenerateContent(
		agent.Context,
		agent.Messages,
		llms.WithTools(agent.Tools),
	)
	if err != nil {
		return nil, err
	}
	// Check for tool calls in the response
	newMessage := llms.TextParts(llms.ChatMessageTypeAI, response.Choices[0].Content)
	for _, tc := range response.Choices[0].ToolCalls {
		newMessage.Parts = append(newMessage.Parts, tc)
	}
	agent.Messages = append(agent.Messages, newMessage)
	return response, nil
}
