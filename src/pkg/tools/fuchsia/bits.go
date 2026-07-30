package fuchsia

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// GetBitsEntryByName gets the entry of a bits declaration by its name
func GetBitsEntryByName(name string, db *database.Database) (database.BitsDecl, error) {
	entry, err := db.GetBitsDecl(name)
	if err != nil {
		return database.BitsDecl{}, err
	}
	return entry, nil
}

// ExecGetBitsTypeByName executes the get_bits_type_by_name tool call
func ExecGetBitsTypeByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetBitsEntryByName(args.Name, th.Db)
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

var GetBitsTypeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_bits_type_by_name",
		Description: "Retrieve the type of a bits declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the bits declaration to retrieve the type for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"name", "rational"},
		},
	},
}

// ExecGetBitsMembersByName executes the get_bits_members_by_name tool call
func ExecGetBitsMembersByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetBitsEntryByName(args.Name, th.Db)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		respContent = resp.Members
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

var GetBitsMembersByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_bits_members_by_name",
		Description: "Retrieve the members of a bits declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the bits declaration to retrieve the members for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"name", "rational"},
		},
	},
}
