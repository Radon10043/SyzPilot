package toy

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// ExecGetCurrentWeather executes the get_current_weather tool call, i.e. call GetCurrentWeather function
// and returns the tool response as llms.MessageContent
func ExecGetCurrentWeather(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Location string `json:"location"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetCurrentWeather(args.Location)
	if err != nil {
		return llms.MessageContent{}, err
	}
	tcResp := llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: tc.ID,
				Name:       tc.FunctionCall.Name,
				Content:    resp,
			},
		},
	}
	return tcResp, nil
}

// GetCurrentWeather is a mock tool function to get current weather info
func GetCurrentWeather(location string) (string, error) {
	info := "The current weather in " + location + " is 72 and sunny."
	b, err := json.Marshal(info)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

var Toys = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "get_current_weather",
			Description: "Get the current weather in a given location",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "The city and state, e.g. San Francisco, CA",
					},
				},
				"required": []string{"location"},
			},
		},
	},
}
