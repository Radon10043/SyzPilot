package tools

import (
	"encoding/json"

	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

// ExecGetEnumCodeByEnumerator execute the get_enum_code_by_enumerator tool call
func ExecGetEnumCodeByEnumerator(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		Enumerator string `json:"enumerator"`
		Rational   string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, err := GetEnumEntryByEnumerator(args.Enumerator)
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

// GetEnumEntry get the enum entry by enumerator
func GetEnumEntryByEnumerator(enumerator string) (database.Enum, error) {
	entry, err := DB.GetEnum(enumerator)
	if err != nil {
		return database.Enum{}, err
	}
	return entry, nil
}

var GetEnumCodeByEnumeratorTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "get_enum_code_by_enumerator",
		Description: "Retrieve the code of an enum given its enumerator.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"enumerator": map[string]interface{}{
					"type":        "string",
					"description": "The name of the enumerator to retrieve the code for.",
				},
				"rational": map[string]interface{}{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"enumerator", "rational"},
		},
	},
}
