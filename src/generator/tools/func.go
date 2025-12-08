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
		Rational     string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetFuncEntryByName(args.FunctionName)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		respContent = resp.Code
	}
	tcResp := llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: tc.ID,
				Name:       tc.FunctionCall.Name,
				Content:    respContent,
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
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"function_name": map[string]any{
					"type":        "string",
					"description": "The name of the function to retrieve the code for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"function_name", "rational"},
		},
	},
}
