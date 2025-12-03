package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// GetTypedefEntryByDefine get the typedef entry by its define
func GetTypedefEntryByDefine(define string, db *database.Database) (database.Typedef, error) {
	entry, err := db.GetTypedef(define)
	if err != nil {
		return database.Typedef{}, err
	}
	return entry, nil
}

// ExecGetTypedefCodeByDefine execute the get_typedef_code_by_define tool call
func ExecGetTypedefCodeByDefine(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		TypedefDefine string `json:"typedef_define"`
		Rational      string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetTypedefEntryByDefine(args.TypedefDefine, DB)
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

var GetTypedefCodeByDefineTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_typedef_code_by_define",
		Description: "Retrieve the code of a typedef given its define.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"typedef_define": map[string]interface{}{
					"type":        "string",
					"description": "The define of the typedef to retrieve the code for.",
				},
				"rational": map[string]interface{}{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"typedef_define", "rational"},
		},
	},
}

// ExecGetTypedefTypeByDefine execute the get_typedef_type_by_define tool call
func ExecGetTypedefTypeByDefine(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		TypedefDefine string `json:"typedef_define"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetTypedefEntryByDefine(args.TypedefDefine, DB)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		respContent = resp.Type
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

var GetTypedefTypeByDefineTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_typedef_type_by_define",
		Description: "Retrieve the type of a typedef given its define.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"typedef_define": map[string]interface{}{
					"type":        "string",
					"description": "The define of the typedef to retrieve the type for.",
				},
			},
			"required": []string{"typedef_define"},
		},
	},
}
