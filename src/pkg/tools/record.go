package tools

import (
	"encoding/json"

	"github.com/Radon10043/SyzPilot/src/pkg/database"
	"github.com/tmc/langchaingo/llms"
)

// GetStructEntryByName get the struct entry by struct name
func GetStructEntryByName(name string, db *database.Database) (database.Record, error) {
	entry, err := db.GetStruct(name)
	if err != nil {
		return database.Record{}, err
	}
	return entry, nil
}

// ExecGetStructCodeByName execute the get_struct_code_by_name tool call
func ExecGetStructCodeByName(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		StructName string `json:"struct_name"`
		Rational   string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetStructEntryByName(args.StructName, th.Db)
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

var GetStructCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_struct_code_by_name",
		Description: "Retrieve the code of a struct given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"struct_name": map[string]any{
					"type":        "string",
					"description": "The name of the struct to retrieve the code for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"struct_name", "rational"},
		},
	},
}

// GetUnionEntryByName get the union entry by union name
func GetUnionEntryByName(name string, db *database.Database) (database.Record, error) {
	entry, err := db.GetUnion(name)
	if err != nil {
		return database.Record{}, err
	}
	return entry, nil
}

// ExecGetUnionCodeByName execute the get_union_code_by_name tool call
func ExecGetUnionCodeByName(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		UnionName string `json:"union_name"`
		Rational  string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetUnionEntryByName(args.UnionName, th.Db)
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

var GetUnionCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_union_code_by_name",
		Description: "Retrieve the code of a union given its name.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"union_name": map[string]any{
					"type":        "string",
					"description": "The name of the union to retrieve the code for.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"union_name", "rational"},
		},
	},
}
