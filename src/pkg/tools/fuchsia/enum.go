package fuchsia

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// GetEnumEntryByName gets the entry of an enum declaration by its name
func GetEnumEntryByName(name string, db *database.Database) (database.EnumDecl, error) {
	entry, err := db.GetEnumDecl(name)
	if err != nil {
		return database.EnumDecl{}, err
	}
	return entry, nil
}

// ExecGetEnumTypeByName executes the get_enum_type_by_name tool call
func ExecGetEnumTypeByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetEnumEntryByName(args.Name, th.Db)
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

var GetEnumTypeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_enum_type_by_name",
		Description: "Retrieve the type of an enum declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the enum declaration to retrieve the type for.",
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

// ExecGetEnumMembersByName executes the get_enum_members_by_name tool call
func ExecGetEnumMembersByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetEnumEntryByName(args.Name, th.Db)
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

var GetEnumMembersByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_enum_members_by_name",
		Description: "Retrieve the members of an enum declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the enum declaration to retrieve the members for.",
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
