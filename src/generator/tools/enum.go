package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// ExecGetEnumCodeByName execute the get_enum_code_by_name tool call
func ExecGetEnumCodeByName(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		EnumName string `json:"enum_name"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetEnumEntryByName(args.EnumName)
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

// GetEnumEntry get the enum entry by enum name
func GetEnumEntryByName(name string) (database.Enum, error) {
	entry, err := DB.GetEnum(name)
	if err != nil {
		return database.Enum{}, err
	}
	return entry, nil
}

var GetEnumCodeByNameTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_enum_code_by_name",
		Description: "Retrieve the code of an enum given its name.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enum_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the enum to retrieve the code for.",
				},
			},
			"required": []string{"enum_name"},
		},
	},
}
