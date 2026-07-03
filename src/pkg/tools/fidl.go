package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// toolResponse builds a tool-call response message with the given text content.
func toolResponse(tc *llms.ToolCall, content string) llms.MessageContent {
	return llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: tc.ID,
				Name:       tc.FunctionCall.Name,
				Content:    content,
			},
		},
	}
}

// FidlSearchDeclarationsTool lets the agent find FIDL declarations by keyword when
// it does not know the exact (possibly mangled or suffixed) declaration name.
var FidlSearchDeclarationsTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name: "fidl_search_declarations",
		Description: "Search the original FIDL definitions for declarations whose name contains " +
			"the given keyword. Use this to discover the correct FIDL declaration behind a syzlang " +
			"node when the exact name is unknown. Returns matching declaration names (syzlang-mangled) " +
			"and their kinds (enum/bits/struct/union/table/protocol/const/alias).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keyword": map[string]any{
					"type":        "string",
					"description": "Substring to search for, case-insensitive (e.g. \"ReadBuffer\" or \"fuchsia_hardware_scsi\").",
				},
				"rationale": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this tool call with these parameters.",
				},
			},
			"required": []string{"keyword", "rationale"},
		},
	},
}

// ExecFidlSearchDeclarations executes the fidl_search_declarations tool call.
func ExecFidlSearchDeclarations(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Keyword   string `json:"keyword"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	if th == nil || th.Fidl == nil {
		return toolResponse(tc, "fidl index is not available"), nil
	}
	const maxResults = 50
	results := th.Fidl.Search(args.Keyword, maxResults)
	var content string
	if len(results) == 0 {
		content = fmt.Sprintf("no declarations found matching %q", args.Keyword)
	} else {
		content = strings.Join(results, "\n")
		if len(results) == maxResults {
			content += fmt.Sprintf("\n... (truncated at %d results, refine the keyword)", maxResults)
		}
	}
	return toolResponse(tc, content), nil
}

// FidlGetDeclarationTool returns the full FIDL definition of a declaration so the
// agent can see the authoritative field names, types, and enum values.
var FidlGetDeclarationTool = llms.Tool{
	Type: "function",
	Function: &llms.FunctionDefinition{
		Name: "fidl_get_declaration",
		Description: "Retrieve the original FIDL definition of a declaration, rendered as readable " +
			"FIDL text (members with their types, enum/bits values, or protocol methods). The name may " +
			"be a FIDL name (\"fuchsia.foo/Bar\"), a syzlang-mangled name (\"fuchsia_foo_Bar\"), or a " +
			"syzlang node name with a generated suffix (e.g. \"...RequestInLine\"). If not found, use " +
			"fidl_search_declarations to locate the right name first.\n\n" +
			"Struct/union/table renderings are annotated with wire-format geometry so you can build the " +
			"InLine/OutOfLine/Handles split and inter-field padding precisely:\n" +
			"- A `// wire_v2: inline_size=.. alignment=.. max_out_of_line=.. max_handles=..` line for the " +
			"whole declaration. When max_out_of_line=0 the generated OutOfLine node is `void void`; when " +
			"max_handles=0 the Handles node is `void void`.\n" +
			"- Each member is tagged with `// @offset size=N [pad=P] [out_of_line<=K] [handles<=H]`. Emit " +
			"`padding array[const[0, int8], P]` after a member whose `pad=P`; give a member an OutOfLine " +
			"field only when it reports `out_of_line`, and a Handles entry only when it reports `handles`.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "The declaration name to retrieve.",
				},
				"rationale": map[string]any{
					"type":        "string",
					"description": "The rationale for choosing this tool call with these parameters.",
				},
			},
			"required": []string{"name", "rationale"},
		},
	},
}

// ExecFidlGetDeclaration executes the fidl_get_declaration tool call.
func ExecFidlGetDeclaration(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error) {
	var args struct {
		Name      string `json:"name"`
		Rationale string `json:"rationale"`
	}
	if err := json.Unmarshal([]byte(tc.FunctionCall.Arguments), &args); err != nil {
		return llms.MessageContent{}, err
	}
	if th == nil || th.Fidl == nil {
		return toolResponse(tc, "fidl index is not available"), nil
	}
	rendered, ok := th.Fidl.Get(args.Name)
	if !ok {
		return toolResponse(tc, fmt.Sprintf(
			"no FIDL declaration found for %q; try fidl_search_declarations with a keyword", args.Name)), nil
	}
	return toolResponse(tc, rendered), nil
}

// FidlToolMap returns the tool map exposing the FIDL query tools to an agent.
func FidlToolMap() map[string]ToolExec {
	return map[string]ToolExec{
		FidlSearchDeclarationsTool.Function.Name: {
			Tool: FidlSearchDeclarationsTool,
			Exec: ExecFidlSearchDeclarations,
		},
		FidlGetDeclarationTool.Function.Name: {
			Tool: FidlGetDeclarationTool,
			Exec: ExecFidlGetDeclaration,
		},
	}
}
