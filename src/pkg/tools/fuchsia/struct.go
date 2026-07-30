package fuchsia

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// GetStructEntryByName gets the entry of a struct declaration by its name
func GetStructEntryByName(name string, db *database.Database) (database.StructDecl, error) {
	entry, err := db.GetStructDecl(name)
	if err != nil {
		return database.StructDecl{}, err
	}
	return entry, nil
}

// ExecGetStructMembersByName executes the get_struct_members_by_name tool call
func ExecGetStructMembersByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetStructEntryByName(args.Name, th.Db)
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

var GetStructMembersByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_struct_members_by_name",
		Description: "Retrieve the members of a struct declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the struct declaration to retrieve the members for.",
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

// ExecGetStructTypeShapeByName executes the get_struct_type_shape_by_name tool call
func ExecGetStructTypeShapeByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetStructEntryByName(args.Name, th.Db)
	var respContent string = ""
	if err != nil {
		respContent = err.Error() // likely not found error
	} else {
		respContent = resp.TypeShape
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

var GetStructTypeShapeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_struct_type_shape_by_name",
		Description: "Retrieve the type shape of a struct declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the struct declaration to retrieve the type shape for.",
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
