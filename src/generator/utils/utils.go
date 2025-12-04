package utils

import (
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

// PartToString converts a llms.ContentPart to its string representation
func PartToString(part llms.ContentPart) string {
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
