package tools

import (
	"encoding/json"
	"strings"

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

// GetMacroDefEntriesByPrefix get macro definition entries by name prefix
func GetMacroDefEntriesByPrefix(prefix string) ([]database.MacroDef, error) {
	entries, err := DB.GetMacroDefByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ExecGetMacroDefCodesByPrefix execute the get_macro_def_codes_by_prefix tool call
func ExecGetMacroDefCodesByPrefix(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		MacroPrefix string `json:"macro_prefix"`
		Rational    string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntriesByPrefix(args.MacroPrefix)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		var codes []string
		for _, md := range resp {
			codes = append(codes, md.Code)
		}
		respContent = strings.Join(codes, "\n")
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

var GetMacroDefCodesByPrefixTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_macro_def_codes_by_prefix",
		Description: "Retrieve the codes of macro definitions given a name prefix.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"macro_prefix": map[string]any{
					"type":        "string",
					"description": "The prefix of the macro definitions to retrieve the codes for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"macro_prefix", "rational"},
		},
	},
}
