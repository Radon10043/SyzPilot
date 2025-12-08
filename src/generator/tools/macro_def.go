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
		Rational  string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntryByName(args.MacroName)
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

var GetMacroDefCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_macro_def_code_by_name",
		Description: "Retrieve the code of a macro definition given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"macro_name": map[string]any{
					"type":        "string",
					"description": "The name of the macro definition to retrieve the code for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"macro_name", "rational"},
		},
	},
}
