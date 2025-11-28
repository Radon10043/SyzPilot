package agent

import (
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/jsonschema"
	"github.com/tmc/langchaingo/llms"
)

func getTomorrowWeather(location string) (string, error) {
	weatherInfo := map[string]interface{}{
		"location":    location,
		"temperature": "72",
		"forecast":    []string{"sunny", "windy"},
	}
	b, err := json.Marshal(weatherInfo)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (agent *Agent) ExecuteToolCalls() (llms.MessageContent, error) {
	latestMessage := agent.Messages[len(agent.Messages)-1]
	toolCalls := []llms.ToolCall{}
	for _, part := range latestMessage.Parts {
		if tc, ok := part.(llms.ToolCall); ok {
			toolCalls = append(toolCalls, tc)
		}
	}
	msg := llms.MessageContent{}
	for _, tc := range toolCalls {
		switch tc.FunctionCall.Name {
		case "getTomorrowWeather":
			var args struct {
				Location string `json:"location"`
			}
			if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
				return llms.MessageContent{}, err
			}
			response, err := getTomorrowWeather(args.Location)
			if err != nil {
				return llms.MessageContent{}, err
			}
			weatherCallResponse := llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: tc.ID,
						Name:       tc.FunctionCall.Name,
						Content:    response,
					},
				},
			}
			msg = weatherCallResponse
		default:
			return llms.MessageContent{}, fmt.Errorf("unsupported tool: %s", tc.FunctionCall.Name)
		}
	}

	return msg, nil
}

var MyTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "getTomorrowWeather",
			Description: "Get the current weather in a given location",
			Parameters: jsonschema.Definition{
				Type: jsonschema.Object,
				Properties: map[string]jsonschema.Definition{
					"rationale": {
						Type:        jsonschema.String,
						Description: "The rationale for choosing this function call with these parameters",
					},
					"location": {
						Type:        jsonschema.String,
						Description: "The city and state, e.g. San Francisco, CA",
					},
					"unit": {
						Type: jsonschema.String,
						Enum: []string{"celsius", "fahrenheit"},
					},
				},
				Required: []string{"rationale", "location"},
			},
		},
	},
}
