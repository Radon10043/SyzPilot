package fuchsia

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

// GetDeclEntryByName gets the entry of a declaration by its name
func GetDeclEntryByName(name string, db *database.Database) (database.Declaration, error) {
	entry, err := db.GetDecl(name)
	if err != nil {
		return database.Declaration{}, err
	}
	return entry, nil
}

// ExecGetDeclByName executes the get_decl_by_name tool call
func ExecGetDeclByName(tc *llms.ToolCall, th *tools.ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name     string `json:"name"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetDeclEntryByName(args.Name, th.Db)
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

var GetDeclByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_decl_by_name",
		Description: "Retrieve the type of a declaration given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the declaration to retrieve the type for.",
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
