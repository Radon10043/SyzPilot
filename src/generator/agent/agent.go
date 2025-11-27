package agent

import (
	"context"

	"github.com/tmc/langchaingo/llms/openai"
)

type Agent struct {
	Context context.Context
	Model   *openai.LLM
}

// Query sends a message to the agent and returns the response
func (agent *Agent) Query(msg string) (string, error) {
	response, err := agent.Model.Call(agent.Context, msg)
	if err != nil {
		return "", err
	}
	return response, nil
}
