package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type jsonSpec struct {
	Include      []string `json:"include"`
	Resources    []string `json:"resource"`
	Define       []string `json:"define"`
	Syscall      []string `json:"syscall"`
	Flags        []string `json:"flags"`
	Struct       []string `json:"struct"`
	Union        []string `json:"union"`
	TypeAlias    []string `json:"type-alias"`
	TypeTemplate []string `json:"type-template"`
	Todo         []string `json:"todo"`
}

// Part2string converts a llms.ContentPart to its string representation
func Part2string(part llms.ContentPart) string {
	switch p := part.(type) {
	case llms.TextContent:
		return p.Text
	case llms.ToolCall:
		return fmt.Sprintf("ID: %s, ToolCall: %s, Args: %v", p.ID, p.FunctionCall.Name, p.FunctionCall.Arguments)
	case llms.ToolCallResponse:
		return fmt.Sprintf("ToolCallID: %s, Content: %s", p.ToolCallID, p.Content)
	default:
		return fmt.Sprintf("unsupported message part type: %T", part)
	}
}

// Json2syzlang converts a JSON specification to syzlang format
func Json2syzlang(jstr string) (string, error) {
	var jspec jsonSpec
	err := json.Unmarshal([]byte(jstr), &jspec)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}
	var sb strings.Builder
	// helper function to write specification
	wrtFunc := func(lines []string, prefix string) {
		if len(lines) == 0 {
			return
		}
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("%s%s\n", prefix, line))
		}
		sb.WriteByte('\n')
	}
	wrtFunc(jspec.Include, "")
	wrtFunc(jspec.Resources, "")
	wrtFunc(jspec.Define, "")
	wrtFunc(jspec.Syscall, "")
	wrtFunc(jspec.Flags, "")
	wrtFunc(jspec.Struct, "")
	wrtFunc(jspec.Union, "")
	wrtFunc(jspec.TypeAlias, "")
	wrtFunc(jspec.TypeTemplate, "")
	wrtFunc(jspec.Todo, "# TODO: ")
	return sb.String(), nil
}
