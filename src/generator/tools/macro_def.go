package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// GetMacroDefEntryByName get the macro definition entry by macro name
func GetMacroDefEntryByName(name string) (database.MacroDef, error) {
	entry, err := DB.GetMacroDef(name)
	if err != nil {
		return database.MacroDef{}, err
	}
	return entry, nil
}

// ExecGetMacroDefCodeByName execute the get_macro_def_code_by_name tool call
func ExecGetMacroDefCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		MacroName string `json:"macro_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntryByName(args.MacroName)
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

var GetMacroDefCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_macro_def_code_by_name",
		Description: "Retrieve the code of a macro definition given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"macro_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the macro definition to retrieve the code for.",
				},
			},
			"required": []string{"macro_name"},
		},
	},
}
