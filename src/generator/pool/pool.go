package pool

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
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
func (s *SpecPool) Remove(elem *SpecElement) error {
	if !s.Exists(elem.Name) {
		return fmt.Errorf("element not found: %s", elem.Name)
	}
	delete(*s, elem.Name)
	return nil
}

// Exists checks if an element with the given name exists in the SpecPool
func (s *SpecPool) Exists(name string) bool {
	_, ok := (*s)[name]
	return ok
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
		return cmp.Compare(priority[a.Type], priority[b.Type])
	})
	return sorted
}

// Syzlang returns the syzlang representation of the SpecPool
func (s *SpecPool) Syzlang() string {
	if s.Empty() {
		return ""
	}
	var (
		syzl strings.Builder
	)
	sorted := s.sort()
	prevType := sorted[0].Type
	for _, elem := range sorted {
		if elem.Type != prevType {
			syzl.WriteString("\n")
		}
		if elem.Type != "include" && elem.Type != "resource" {
			comment := fmt.Sprintf("# valid: %t\n", elem.Valid)
			syzl.WriteString(comment)
		}
		prevType = elem.Type
		syzl.WriteString(elem.Code + "\n")
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
