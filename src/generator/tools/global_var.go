package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// GetGlobalVarEntry get the global variable entry by variable name
func GetGlobalVarEntryByName(name string) (database.GlobalVar, error) {
	entry, err := DB.GetGlobalVar(name)
	if err != nil {
		return database.GlobalVar{}, err
	}
	return entry, nil
}

// ExecGetGlobalVarCodeByName execute the get_global_var_code_by_name tool call
func ExecGetGlobalVarCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		VarName string `json:"var_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetGlobalVarEntryByName(args.VarName)
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

var GetGlobalVarCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_global_var_code_by_name",
		Description: "Retrieve the code of a global variable given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"var_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the global variable to retrieve the code for.",
				},
			},
			"required": []string{"var_name"},
		},
	},
}
