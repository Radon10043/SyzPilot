package tools

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/llms"
)

// CheckSpecValidity check the validity of the given specification
func CheckSpecValidity(spec string) (string, error) {
	SC.CleanWorkdir()
	path := filepath.Join(SC.Workdir, "sys", "linux", "spec.txt")
	os.WriteFile(path, []byte(spec), 0644)
	_, stderr, err := SC.ExtractConst()
	if err != nil {
		return stderr.String(), err
	}
	_, stderr, err = SC.CheckValidity()
	if err != nil {
		return stderr.String(), err
	}
	return "Specification is valid.", nil
}

// ExecCheckSpecValidity execute the check_spec_validity tool call
func ExecCheckSpecValidity(tc llms.ToolCall) (llms.MessageContent, error) {
	var args struct {
		Spec string `json:"spec"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, _ := CheckSpecValidity(args.Spec)
	tcResp := llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: tc.ID,
				Name:       tc.FunctionCall.Name,
				Content:    resp,
			},
		},
	}
	return tcResp, nil
}

var CheckSpecValidityTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name:        "check_spec_validity",
		Description: "Check the validity of the given specification.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"spec": map[string]interface{}{
					"type":        "string",
					"description": "The specification to check.",
				},
			},
			"required": []string{"spec"},
		},
	},
}
