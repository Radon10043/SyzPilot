package tools

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/tmc/langchaingo/llms"
)

// CheckSpecValidity check the validity of the given specification
func CheckSpecValidity(spec string, sc *check.SyzCheck) (string, error) {
	sc.CleanWorkdir()
	path := filepath.Join(sc.Workdir, "sys", "linux", "spec.txt")
	os.WriteFile(path, []byte(spec), 0644)
	_, stderr, err := sc.ExtractConst()
	if err != nil {
		return stderr.String(), err
	}
	_, stderr, err = sc.CheckValidity()
	if err != nil {
		return stderr.String(), err
	}
	return "Specification is valid.", nil
}

// ExecCheckSpecValidity execute the check_spec_validity tool call
func ExecCheckSpecValidity(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Spec     string `json:"spec"`
		Rational string `json:"rational"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	resp, _ := CheckSpecValidity(args.Spec, th.Sc)
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
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"spec": map[string]any{
					"type":        "string",
					"description": "The specification to check.",
				},
				"rational": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this function call with these parameters",
				},
			},
			"required": []string{"spec", "rational"},
		},
	},
}
