package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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
	wrtFunc := func(lines []string, prefix string) { // helper function to convert json spec to syz spec
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
