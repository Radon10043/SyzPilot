package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// GetFuncEntry get the function entry by function name
func GetFuncEntryByName(name string) (database.Function, error) {
	entry, err := DB.GetFunction(name)
	if err != nil {
		return database.Function{}, err
	}
	return entry, nil
}

// ExecGetFuncCodeByName execute the get_func_code_by_name tool call
func ExecGetFuncCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		FunctionName string `json:"function_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetFuncEntryByName(args.FunctionName)
	if err != nil {
		return llms.MessageContent{}, err
	}
	tcResp := llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: tc.ID,
				Name:       tc.FunctionCall.Name,
				Content:    resp.Code,
			},
		},
	}
	return tcResp, nil
}

var GetFuncCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_func_code_by_name",
		Description: "Retrieve the code of a function given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"function_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the function to retrieve the code for.",
				},
			},
			"required": []string{"function_name"},
		},
	},
}
