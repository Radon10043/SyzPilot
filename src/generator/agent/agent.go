package agent

import (
	"context"
	"log"

	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	Ctx      context.Context       // context for llm operations
	Model    *openai.LLM           // model instance
	Tools    []llms.Tool           // available tools
	Messages []llms.MessageContent // message history
}

// CleanMessages clear the message history of the agent
func (a *Agent) CleanMessages() {
	a.Messages = []llms.MessageContent{}
}

// AddHumanMessage appends a human message to the agent's message history
func (a *Agent) AddHumanMessage(input string) {
	msg := llms.TextParts(llms.ChatMessageTypeHuman, input)
	a.Messages = append(a.Messages, msg)
}

// Query query the llm with the current messages and update the history
func (a *Agent) Query() (*llms.ContentResponse, error) {
	// query the llm with existing messages
	response, err := a.Model.GenerateContent(a.Ctx, a.Messages, llms.WithTools(a.Tools))
	if err != nil {
		return nil, err
	}
	// update the message history with ai's response
	choice := response.Choices[0]
	aiResponse := llms.TextParts(llms.ChatMessageTypeAI, choice.Content)
	for _, tc := range choice.ToolCalls {
		aiResponse.Parts = append(aiResponse.Parts, tc)
	}
	a.Messages = append(a.Messages, aiResponse)
	return response, nil
}

// ExecTools execute the tools called by the llm and update the message history
func (a *Agent) ExecTools() error {
	lastMsg := a.Messages[len(a.Messages)-1]
	for _, part := range lastMsg.Parts {
		tc, ok := part.(llms.ToolCall)
		if !ok {
			continue
		}
		switch tc.FunctionCall.Name {
		case "get_current_weather":
			tcResp, err := myTools.ExecGetCurrentWeather(tc)
			if err != nil {
				return err
			}
			a.Messages = append(a.Messages, tcResp)
		default:
			log.Printf("unknown tool: %s", tc.FunctionCall.Name)
		}
	}
	return nil
}
