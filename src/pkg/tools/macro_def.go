package tools

import (
	"encoding/json"
	"strings"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/tmc/langchaingo/llms"
)

// GetMacroDefEntryByName get the macro definition entry by macro name
func GetMacroDefEntryByName(name string, db *database.Database) (database.MacroDef, error) {
	entry, err := db.GetMacroDefByName(name)
	if err != nil {
		return database.MacroDef{}, err
	}
	return entry, nil
}

// ExecGetMacroDefCodeByName execute the get_macro_def_code_by_name tool call
func ExecGetMacroDefCodeByName(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		MacroName string `json:"macro_name"`
		Rational  string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntryByName(args.MacroName, th.Db)
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

// ExecGetMacroDefLocByName execute the get_macro_def_loc_by_name tool call
func ExecGetMacroDefLocByName(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		MacroName string `json:"macro_name"`
		Rational  string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntryByName(args.MacroName, th.Db)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		respContent = resp.File + ":" + string(rune(resp.Line))
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

var GetMacroDefLocByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_macro_def_loc_by_name",
		Description: "Retrieve the file location of a macro definition given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"macro_name": map[string]any{
					"type":        "string",
					"description": "The name of the macro definition to retrieve the location for.",
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

// GetMacroDefEntriesByPattern get macro definition entries by name pattern
func GetMacroDefEntriesByPattern(pattern string, db *database.Database) ([]database.MacroDef, error) {
	entries, err := db.GetMacroDefByPattern(pattern)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// ExecGetMacroDefCodesByPattern execute the get_macro_def_codes_by_pattern tool call
func ExecGetMacroDefCodesByPattern(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		MacroPattern string `json:"macro_pattern"`
		Rational     string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetMacroDefEntriesByPattern(args.MacroPattern, th.Db)
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

var GetMacroDefCodesByPatternTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name: "get_macro_def_codes_by_pattern",
		Description: "Retrieve the source codes of macro definitions whose names match a specific pattern using SQLite GLOB syntax. " +
			"IMPORTANT: This is CASE-SENSITIVE. Use Unix-style wildcards: '*' (matches any sequence), '?' (matches single char), and '[...]' (matches char set). " +
			"Do NOT use SQL '%' syntax.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"macro_pattern": map[string]any{
					"type":        "string",
					"description": "The pattern of the macro definitions to retrieve the codes for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"macro_pattern", "rational"},
		},
	},
}
