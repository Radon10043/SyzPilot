package utils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

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

// ExtractFirstCodeBlock extracts the first fenced code block with the specified language from the given Markdown content.
func ExtractFirstCodeBlock(markdownContent string, targetLang string) (string, bool) {
	src := []byte(markdownContent)
	md := goldmark.New()
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)
	code := ""
	found := false
	walkFunc := func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || found {
			return ast.WalkContinue, nil
		}
		if n.Kind() != ast.KindFencedCodeBlock {
			return ast.WalkContinue, nil
		}
		codeBlock := n.(*ast.FencedCodeBlock)
		lang := string(codeBlock.Language(src))
		if !strings.EqualFold(lang, targetLang) {
			return ast.WalkContinue, nil
		}
		var buf bytes.Buffer
		lines := codeBlock.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(src))
		}
		code = buf.String()
		found = true
		return ast.WalkStop, nil
	}
	ast.Walk(doc, walkFunc)
	return code, found
}
