package pool

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/syzkaller/pkg/ast"
)

// SpecElement represents an element in the specification queue
type SpecElement struct {
	// element name, e.g. "openat$ppp"
	Name string `json:"name"`
	// element type, include "include", "resource", "define", "init_syscall",
	// "syscall", "flag", "struct", "union", "type-alias", "type-template"
	Type string `json:"type"`
	// element code snippet in syzlang
	Code string `json:"code"`
	// whether the element is valid, i.e. passes syz-extract & syz-check
	Valid bool `json:"valid"`
}

type specElementOptions func(*SpecElement)

// WithName sets the Name field of a SpecElement
func WithName(name string) specElementOptions {
	return func(e *SpecElement) {
		e.Name = name
	}
}

// WithType sets the Type field of a SpecElement
func WithType(typ string) specElementOptions {
	return func(e *SpecElement) {
		e.Type = typ
	}
}

// WithCode sets the Code field of a SpecElement
func WithCode(code string) specElementOptions {
	return func(e *SpecElement) {
		e.Code = code
	}
}

// WithValid sets the Valid field of a SpecElement
func WithValid(valid bool) specElementOptions {
	return func(e *SpecElement) {
		e.Valid = valid
	}
}

// NewSpecElement creates a new SpecElement with the given options
func NewSpecElement(opts ...specElementOptions) SpecElement {
	elem := SpecElement{}
	for _, opt := range opts {
		opt(&elem)
	}
	return elem
}

// newSpecElementFromSyzNode converts a syzlang AST node to a SpecElement
func newSpecElementFromSyzNode(node ast.Node) (SpecElement, error) {
	d := &ast.Description{
		Nodes: []ast.Node{node},
	}
	b := ast.Format(d)
	s := strings.TrimSpace(string(b))
	switch o := node.(type) {
	case *ast.Include:
		return SpecElement{
			Type: "include",
			Name: o.File.Value,
			Code: s,
			// If a syzlang spec can be parsed successfully, it must be valid
			Valid: true,
		}, nil
	case *ast.Resource:
		return SpecElement{
			Type:  "resource",
			Name:  o.Name.Name,
			Code:  s,
			Valid: true,
		}, nil
	case *ast.Define:
		return SpecElement{
			Type:  "define",
			Name:  o.Name.Name,
			Code:  s,
			Valid: true,
		}, nil
	case *ast.Call:
		return SpecElement{
			Type:  "syscall",
			Name:  o.Name.Name,
			Code:  s,
			Valid: true,
		}, nil
	case *ast.IntFlags:
		return SpecElement{
			Type:  "flag",
			Name:  o.Name.Name,
			Code:  s,
			Valid: true,
		}, nil
	case *ast.StrFlags:
		return SpecElement{
			Type:  "flag",
			Name:  o.Name.Name,
			Code:  s,
			Valid: true,
		}, nil
	case *ast.Struct:
		if o.IsUnion {
			return SpecElement{
				Type:  "union",
				Name:  o.Name.Name,
				Code:  s,
				Valid: true,
			}, nil
		} else {
			return SpecElement{
				Type:  "struct",
				Name:  o.Name.Name,
				Code:  s,
				Valid: true,
			}, nil
		}
	case *ast.TypeDef:
		if o.Struct != nil {
			return SpecElement{
				Type:  "type-template",
				Name:  o.Name.Name,
				Code:  s,
				Valid: true,
			}, nil
		} else {
			return SpecElement{
				Type:  "type-alias",
				Name:  o.Name.Name,
				Code:  s,
				Valid: true,
			}, nil
		}
	case *ast.Comment, *ast.NewLine, *ast.Meta, *ast.Incdir:
		return SpecElement{}, nil
	default:
		return SpecElement{}, fmt.Errorf("unknown syzlang node type: %T", node)
	}
}

// SpecPool represents a set of SpecElements, deduplicated by Name
type SpecPool map[string]*SpecElement

// Len returns the number of elements in the SpecPool
func (s *SpecPool) Len() int {
	return len(*s)
}

// Empty checks if the SpecPool is empty
func (s *SpecPool) Empty() bool {
	return len(*s) == 0
}

// Clear removes all elements from the SpecPool
func (s *SpecPool) Clear() {
	clear(*s)
}

// Insert adds an element to the SpecPool
func (s *SpecPool) Insert(elem SpecElement) {
	(*s)[elem.Name] = &elem
}

func (s *SpecPool) Get(name string) (SpecElement, error) {
	if !s.Exists(name) {
		return SpecElement{}, fmt.Errorf("element not found: %s", name)
	}
	return *(*s)[name], nil
}

// Remove removes and returns an element from the SpecPool by name
func (s *SpecPool) Remove(name string) error {
	if !s.Exists(name) {
		return fmt.Errorf("element not found: %s", name)
	}
	delete(*s, name)
	return nil
}

// Exists checks if an element with the given name exists in the SpecPool
func (s *SpecPool) Exists(name string) bool {
	_, ok := (*s)[name]
	return ok
}

// Intersect returns a new SpecPool containing elements that exist in both SpecPools
func (s *SpecPool) Intersect(other *SpecPool) *SpecPool {
	intersection := NewSpecPool()
	for name, elem := range *s {
		if other.Exists(name) {
			intersection.Insert(*elem)
		}
	}
	return intersection
}

// Difference returns a new SpecPool containing elements that exist in the current
// SpecPool but not in the other SpecPool
func (s *SpecPool) Difference(other *SpecPool) *SpecPool {
	difference := NewSpecPool()
	for name, elem := range *s {
		if !other.Exists(name) {
			difference.Insert(*elem)
		}
	}
	return difference
}

// Merge merges another SpecPool into the current SpecPool, with the other SpecPool
// taking precedence in case of conflicts
func (s *SpecPool) Merge(other *SpecPool) {
	maps.Copy((*s), *other)
}

// SyncType syncs the types of elements in the SpecPool with another SpecPool.
// We mainly use this function to sync init_syscall.
func (s1 *SpecPool) SyncType(s2 *SpecPool) {
	for k1, v1 := range *s1 {
		if v2, ok := (*s2)[k1]; ok {
			v1.Type = v2.Type
			(*s1)[k1] = v1
		}
	}
}

// Json returns the JSON representation of the SpecPool
func (s *SpecPool) Json() (string, error) {
	jbytes, err := json.MarshalIndent(*s, "", "\t")
	if err != nil {
		return "", err
	}
	return string(jbytes), nil
}

// sort sorts the SpecElements in SpecPool based on the priority of element types
func (s SpecPool) sort() []SpecElement {
	priority := map[string]int{
		"include":       0,
		"resource":      1,
		"define":        2,
		"init_syscall":  3,
		"syscall":       4,
		"flag":          5,
		"struct":        6,
		"union":         7,
		"type-alias":    8,
		"type-template": 9,
	}
	sorted := make([]SpecElement, 0, len(s))
	for _, elem := range s {
		sorted = append(sorted, *elem)
	}
	slices.SortFunc(sorted, func(a, b SpecElement) int {
		if c := cmp.Compare(priority[a.Type], priority[b.Type]); c != 0 {
			return c
		}
		// break ties by name so the output order is deterministic
		return cmp.Compare(a.Name, b.Name)
	})
	return sorted
}

type syzlangConfig struct {
	showValidComment bool
}

type SyzlangOption func(*syzlangConfig)

// WithValidComment configures whether to show valid comments in the syzlang output
func WithValidComment(show bool) SyzlangOption {
	return func(cfg *syzlangConfig) {
		cfg.showValidComment = show
	}
}

// Syzlang returns the syzlang representation of the SpecPool
func (s *SpecPool) Syzlang(opts ...SyzlangOption) string {
	if s.Empty() {
		return ""
	}
	var (
		syzl strings.Builder
		cfg  syzlangConfig = syzlangConfig{
			showValidComment: false,
		}
	)
	for _, opt := range opts {
		opt(&cfg)
	}
	sorted := s.sort()
	prevType := sorted[0].Type
	for _, elem := range sorted {
		if elem.Type != prevType {
			syzl.WriteString("\n")
		}
		if elem.Type != "include" && elem.Type != "resource" && cfg.showValidComment {
			comment := fmt.Sprintf("# valid: %t\n", elem.Valid)
			syzl.WriteString(comment)
		}
		prevType = elem.Type
		syzl.WriteString(elem.Code)
		syzl.WriteString("\n")
	}
	return syzl.String()
}

// NewSpecPool creates and returns a new SpecPool
func NewSpecPool() *SpecPool {
	return &SpecPool{}
}

// NewSpecPoolFromJson creates a SpecPool from a JSON string
func NewSpecPoolFromJson(jstr string) (*SpecPool, error) {
	s := NewSpecPool()
	err := json.Unmarshal([]byte(jstr), &s)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func NewSpecPoolFromSyzlang(syzl string) (*SpecPool, error) {
	// parse syzlang spec as ast
	var err error
	desc := ast.Parse([]byte(syzl), "spec.txt", func(pos ast.Pos, msg string) {
		err = fmt.Errorf("syzlang syntax error at %v: %v", pos, msg)
	})
	if err != nil {
		return nil, err
	}

	// traver AST
	// NOTE: some info may be lost after conversion, e.g. comment, meta, etc.
	// But it should not be impact spec generation.
	spool := NewSpecPool()
	for _, node := range desc.Nodes {
		se, err := newSpecElementFromSyzNode(node)
		if err != nil {
			return nil, err
		}
		if se == (SpecElement{}) {
			continue
		}
		spool.Insert(se)
	}
	return spool, nil
}
