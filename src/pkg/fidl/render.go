package fidl

import (
	"fmt"
	"strings"
)

// renderType renders a FIDL type object into its canonical FIDL textual form,
// e.g. "uint32", "vector<fuchsia.foo/Bar>:64", "array<uint8, 8>", "client_end:fuchsia.foo/Baz".
// Identifier types keep their fully-qualified FIDL name so the agent can look them
// up with the fidl_get_declaration tool.
func renderType(t *fidlType) string {
	if t == nil {
		return "void"
	}
	opt := ""
	if t.Nullable {
		opt = "?"
	}
	switch t.Kind {
	case "primitive", "internal":
		return t.Subtype
	case "identifier":
		return t.Identifier + opt
	case "string":
		if t.MaybeElementCount != nil {
			return fmt.Sprintf("string:%d%s", *t.MaybeElementCount, opt)
		}
		return "string" + opt
	case "vector":
		if t.MaybeElementCount != nil {
			return fmt.Sprintf("vector<%s>:%d%s", renderType(t.ElementType), *t.MaybeElementCount, opt)
		}
		return fmt.Sprintf("vector<%s>%s", renderType(t.ElementType), opt)
	case "array":
		n := 0
		if t.ElementCount != nil {
			n = *t.ElementCount
		}
		return fmt.Sprintf("array<%s, %d>", renderType(t.ElementType), n)
	case "handle":
		if t.Subtype != "" && t.Subtype != "handle" {
			return "handle<" + t.Subtype + ">" + opt
		}
		return "handle" + opt
	case "endpoint":
		role := "server_end"
		if t.Role == "client" {
			role = "client_end"
		}
		return role + ":" + t.Protocol + opt
	default:
		if t.Subtype != "" {
			return t.Subtype
		}
		return t.Kind
	}
}

func (d constDecl) render() string {
	val := ""
	if d.Value != nil {
		val = d.Value.Value
		if d.Value.Expression != "" {
			val = d.Value.Expression
		}
	}
	return fmt.Sprintf("// kind: const\nconst %s %s = %s\n", d.Name, renderType(d.Type), val)
}

func (d enumDecl) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "// kind: enum\nenum %s : %s {\n", d.Name, d.Type)
	for _, m := range d.Members {
		fmt.Fprintf(&b, "    %s = %s\n", m.Name, memberValue(m))
	}
	b.WriteString("}\n")
	return b.String()
}

func (d bitsDecl) render() string {
	var b strings.Builder
	underlying := "uint32"
	if d.Type != nil {
		underlying = renderType(d.Type)
	}
	fmt.Fprintf(&b, "// kind: bits\nbits %s : %s {\n", d.Name, underlying)
	for _, m := range d.Members {
		fmt.Fprintf(&b, "    %s = %s\n", m.Name, memberValue(m))
	}
	b.WriteString("}\n")
	return b.String()
}

func (d recordDecl) render(kind string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// kind: %s\n", kind)
	if d.TypeShape != nil {
		// The declaration-level geometry tells the agent, at a glance, whether the
		// generated OutOfLine node is empty (max_out_of_line == 0 -> `void void`) and
		// whether the Handles node is empty (max_handles == 0 -> `void void`).
		fmt.Fprintf(&b, "// wire_v2: inline_size=%d alignment=%d max_out_of_line=%d max_handles=%d\n",
			d.TypeShape.InlineSize, d.TypeShape.Alignment, d.TypeShape.MaxOutOfLine, d.TypeShape.MaxHandles)
	}
	fmt.Fprintf(&b, "%s %s {\n", kind, d.Name)
	for _, m := range d.Members {
		if m.Ordinal != 0 { // union/table members carry an ordinal
			fmt.Fprintf(&b, "    %d: %s %s%s\n", m.Ordinal, m.Name, renderType(m.Type), shapeComment(m))
		} else {
			fmt.Fprintf(&b, "    %s %s%s\n", m.Name, renderType(m.Type), shapeComment(m))
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// shapeComment renders a trailing "// ..." annotation describing a member's wire
// geometry: its offset within the inline struct, its inline size, any trailing
// padding, and whether it contributes out-of-line bytes or handles. This is the
// information the agent needs to build the InLine/OutOfLine/Handles split and the
// `padding array[const[0, int8], N]` fillers. It returns "" when no shape data is
// available (older IR without *_shape_v2), so rendering degrades cleanly.
func shapeComment(m member) string {
	var parts []string
	if m.FieldShape != nil {
		parts = append(parts, fmt.Sprintf("@%d", m.FieldShape.Offset))
	}
	if m.Type != nil && m.Type.TypeShape != nil {
		parts = append(parts, fmt.Sprintf("size=%d", m.Type.TypeShape.InlineSize))
	}
	if m.FieldShape != nil && m.FieldShape.Padding > 0 {
		parts = append(parts, fmt.Sprintf("pad=%d", m.FieldShape.Padding))
	}
	if m.Type != nil && m.Type.TypeShape != nil {
		if m.Type.TypeShape.MaxOutOfLine > 0 {
			parts = append(parts, fmt.Sprintf("out_of_line<=%d", m.Type.TypeShape.MaxOutOfLine))
		}
		if m.Type.TypeShape.MaxHandles > 0 {
			parts = append(parts, fmt.Sprintf("handles<=%d", m.Type.TypeShape.MaxHandles))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\t// " + strings.Join(parts, " ")
}

func (d aliasDecl) render(kind string) string {
	return fmt.Sprintf("// kind: %s\n%s %s = %s\n", kind, kind, d.Name, renderType(d.Type))
}

func (d protocolDecl) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "// kind: protocol\nprotocol %s {\n", d.Name)
	for _, m := range d.Methods {
		req := ""
		if m.HasRequest && m.MaybeRequestPayload != nil {
			req = renderType(m.MaybeRequestPayload)
		}
		switch {
		case m.HasResponse && m.MaybeResponsePayload != nil:
			fmt.Fprintf(&b, "    %s(%s) -> (%s)\n", m.Name, req, renderType(m.MaybeResponsePayload))
		case m.HasResponse:
			fmt.Fprintf(&b, "    %s(%s) -> ()\n", m.Name, req)
		default:
			fmt.Fprintf(&b, "    %s(%s)\n", m.Name, req) // one-way / event
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// memberValue prefers the original expression (e.g. "0x01") and falls back to the
// resolved decimal value.
func memberValue(m member) string {
	if m.Value == nil {
		return "?"
	}
	if m.Value.Expression != "" {
		return m.Value.Expression
	}
	return m.Value.Value
}
