package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// GetStructEntryByName get the struct entry by struct name
func GetStructEntryByName(name string) (database.Record, error) {
	entry, err := DB.GetStruct(name)
	if err != nil {
		return database.Record{}, err
	}
	return entry, nil
}

// ExecGetStructCodeByName execute the get_struct_code_by_name tool call
func ExecGetStructCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		StructName string `json:"struct_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetStructEntryByName(args.StructName)
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

var GetStructCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_struct_code_by_name",
		Description: "Retrieve the code of a struct given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"struct_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the struct to retrieve the code for.",
				},
			},
			"required": []string{"struct_name"},
		},
	},
}

// GetUnionEntryByName get the union entry by union name
func GetUnionEntryByName(name string) (database.Record, error) {
	entry, err := DB.GetUnion(name)
	if err != nil {
		return database.Record{}, err
	}
	return entry, nil
}

// ExecGetUnionCodeByName execute the get_union_code_by_name tool call
func ExecGetUnionCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		UnionName string `json:"union_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetUnionEntryByName(args.UnionName)
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

var GetUnionCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_union_code_by_name",
		Description: "Retrieve the code of a union given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"union_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the union to retrieve the code for.",
				},
			},
			"required": []string{"union_name"},
		},
	},
}
