package ast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/syzkaller/pkg/ast"
)

type JsonSpec struct {
	Include      []string `json:"include"`
	Resource     []string `json:"resource"`
	Define       []string `json:"define"`
	Syscall      []string `json:"syscall"`
	Flags        []string `json:"flags"`
	Struct       []string `json:"struct"`
	Union        []string `json:"union"`
	TypeAlias    []string `json:"type-alias"`
	TypeTemplate []string `json:"type-template"`
	Todo         []string `json:"todo"`
}

func (js *JsonSpec) Clone() *JsonSpec {
	copySlice := func(src []string) []string {
		dst := make([]string, len(src))
		copy(dst, src)
		return dst
	}
	njs := &JsonSpec{}
	njs.Include = copySlice(js.Include)
	njs.Resource = copySlice(js.Resource)
	njs.Define = copySlice(js.Define)
	njs.Syscall = copySlice(js.Syscall)
	njs.Flags = copySlice(js.Flags)
	njs.Struct = copySlice(js.Struct)
	njs.Union = copySlice(js.Union)
	njs.TypeAlias = copySlice(js.TypeAlias)
	njs.TypeTemplate = copySlice(js.TypeTemplate)
	njs.Todo = copySlice(js.Todo)
	return njs
}

type Option func(*JsonSpec)

// WithInclude sets the Include field of JsonSpec
func WithInclude(lines []string) Option {
	return func(js *JsonSpec) {
		js.Include = lines
	}
}

// WithDefine sets the Define field of JsonSpec
func WithResource(lines []string) Option {
	return func(js *JsonSpec) {
		js.Resource = lines
	}
}

// WithDefine sets the Define field of JsonSpec
func WithDefine(lines []string) Option {
	return func(js *JsonSpec) {
		js.Define = lines
	}
}

// WithSyscall sets the Syscall field of JsonSpec
func WithSyscall(lines []string) Option {
	return func(js *JsonSpec) {
		js.Syscall = lines
	}
}

// WithFlags sets the Flags field of JsonSpec
func WithFlags(lines []string) Option {
	return func(js *JsonSpec) {
		js.Flags = lines
	}
}

// WithStruct sets the Struct field of JsonSpec
func WithStruct(lines []string) Option {
	return func(js *JsonSpec) {
		js.Struct = lines
	}
}

// WithUnion sets the Union field of JsonSpec
func WithUnion(lines []string) Option {
	return func(js *JsonSpec) {
		js.Union = lines
	}
}

// WithTypeAlias sets the TypeAlias field of JsonSpec
func WithTypeAlias(lines []string) Option {
	return func(js *JsonSpec) {
		js.TypeAlias = lines
	}
}

// WithTypeTemplate sets the TypeTemplate field of JsonSpec
func WithTypeTemplate(lines []string) Option {
	return func(js *JsonSpec) {
		js.TypeTemplate = lines
	}
}

// WithTodo sets the Todo field of JsonSpec
func WithTodo(lines []string) Option {
	return func(js *JsonSpec) {
		js.Todo = lines
	}
}

// NewJsonSpec creates a new JsonSpec instance
func NewJsonSpec(opts ...Option) *JsonSpec {
	js := &JsonSpec{
		Include:      []string{},
		Resource:     []string{},
		Define:       []string{},
		Syscall:      []string{},
		Flags:        []string{},
		Struct:       []string{},
		Union:        []string{},
		TypeAlias:    []string{},
		TypeTemplate: []string{},
		Todo:         []string{},
	}
	for _, opt := range opts {
		opt(js)
	}
	return js
}

// addSyzNode adds a syzlang AST node to the appropriate field in JsonSpec
func (js *JsonSpec) addSyzNode(node ast.Node) error {
	d := &ast.Description{
		Nodes: []ast.Node{node},
	}
	b := ast.Format(d)
	s := strings.TrimSpace(string(b))
	switch o := node.(type) {
	case *ast.Include:
		js.Include = append(js.Include, s)
	case *ast.Resource:
		js.Resource = append(js.Resource, s)
	case *ast.Define:
		js.Define = append(js.Define, s)
	case *ast.Call:
		js.Syscall = append(js.Syscall, s)
	case *ast.IntFlags, *ast.StrFlags:
		js.Flags = append(js.Flags, s)
	case *ast.Struct:
		if o.IsUnion {
			js.Union = append(js.Union, s)
		} else {
			js.Struct = append(js.Struct, s)
		}
	case *ast.TypeDef:
		if o.Struct != nil {
			js.TypeTemplate = append(js.TypeTemplate, s)
		} else {
			js.TypeAlias = append(js.TypeAlias, s)
		}
	case *ast.Comment:
		if s, ok := strings.CutPrefix(o.Text, " TODO: "); ok {
			js.Todo = append(js.Todo, s)
		}
	case *ast.NewLine:
	case *ast.Meta:
	default:
		return fmt.Errorf("unknown syzlang node type: %T", node)
	}
	return nil
}

// Json2syzlang converts a JSON specification to syzlang format
func Json2syzlang(jstr string) (string, error) {
	var jspec JsonSpec
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
	wrtFunc(jspec.Resource, "")
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

func Syzlang2JsonSpec(spec string) (*JsonSpec, error) {
	// parse syzlang spec as ast
	var err error
	desc := ast.Parse([]byte(spec), "spec.txt", func(pos ast.Pos, msg string) {
		err = fmt.Errorf("syzlang syntax error at %v: %v", pos, msg)
	})
	if err != nil {
		return nil, err
	}

	// traverse ast
	jspec := NewJsonSpec()
	for _, node := range desc.Nodes {
		jspec.addSyzNode(node)
	}
	return jspec, nil
}

// Syzlang2json converts a syzlang specification to JSON format
func Syzlang2json(spec string) (string, error) {
	jspec, err := Syzlang2JsonSpec(spec)
	if err != nil {
		return "", err
	}
	var jbytes bytes.Buffer
	encoder := json.NewEncoder(&jbytes)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(jspec)
	if err != nil {
		return "", fmt.Errorf("failed to encode JSON: %w", err)
	}
	return jbytes.String(), nil
}
