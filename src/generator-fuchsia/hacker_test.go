package main

import (
	"strings"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/pool"
	"github.com/google/uuid"
)

func TestHackExpandsCaptureGroups(t *testing.T) {
	hacker := Hacker{
		HackPatterns: map[string]func(string) string{
			`(?m)^(\s+)resource(\s+)`: func(_ string) string {
				return "${1}reso${2}"
			},
			`(?m)^(\s+)fidl_union_member\[(\d+),(\s+)\]`: func(_ string) string {
				return "${1}member fidl_union_member[$2, array[int8, 1]]"
			},
		},
	}

	got := hacker.Hack("header\n\tresource handle\n  fidl_union_member[42, ]\n")
	want := "header\n\treso handle\n  member fidl_union_member[42, array[int8, 1]]\n"
	if got != want {
		t.Fatalf("Hack() mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestHackRunsReplacementForEveryMatch(t *testing.T) {
	replacements := 0
	hacker := Hacker{
		HackPatterns: map[string]func(string) string{
			`array\[\]`: func(_ string) string {
				replacements++
				return "array[int8, 1]"
			},
		},
	}

	got := hacker.Hack("array[] array[]")
	if got != "array[int8, 1] array[int8, 1]" {
		t.Fatalf("Hack() = %q", got)
	}
	if replacements != 2 {
		t.Fatalf("replacement function called %d times, want 2", replacements)
	}
	if strings.Contains(got, "array[]") {
		t.Fatalf("Hack() left an unmatched array: %q", got)
	}
}

func TestUnhackFidlUnionMemberAfterPoolFormatting(t *testing.T) {
	hacker := NewHacker(
		WithHackPatterns(map[string]func(string) string{
			`array\[\]`:                   func(string) string { return "array[int8, 114514]" },
			`(?m)^(\s{4})resource(\s{1})`: func(string) string { return "${1}reso${2}" },
			`(?m)^(?:\s{5})fidl_union_member\[(\d+),(?:\s+)\]`: func(string) string {
				return "    yjsp_" + uuid.New().String()[:8] + " fidl_union_member[$1, array[int8, 1919810]]"
			},
		}),
		WithUnhackPatterns(map[string]func(string) string{
			`array\[int8, 114514\]`: func(string) string { return "array[]" },
			`(?m)^([ \t]*)yjsp_[a-zA-Z0-9_-]+[ \t]+fidl_union_member\[(\d+),[ \t]*array\[int8,[ \t]*1919810\]\][ \t]*$`: func(string) string {
				return "${1}fidl_union_member[$2, ]"
			},
		}),
	)
	original := `example {
     fidl_union_member[42, ]
}`

	hacked := hacker.Hack(original)
	if !strings.Contains(hacked, "yjsp_") {
		t.Fatalf("Hack() did not create placeholder field:\n%s", hacked)
	}
	po, err := pool.NewSpecPoolFromSyzlang(hacked)
	if err != nil {
		t.Fatalf("NewSpecPoolFromSyzlang() failed: %v", err)
	}

	hacker.UnhackSpecPool(po)
	elem, err := po.Get("example")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if strings.Contains(elem.Code, "yjsp_") {
		t.Fatalf("UnhackSpecPool() left placeholder field:\n%s", elem.Code)
	}
	if !strings.Contains(elem.Code, "fidl_union_member[42, ]") {
		t.Fatalf("UnhackSpecPool() did not restore union member:\n%s", elem.Code)
	}
}

func TestUnhackFidlUnionMemberAcceptsFormattedWhitespace(t *testing.T) {
	hacker := NewHacker(
		WithHackPatterns(map[string]func(string) string{
			`array\[\]`:                   func(string) string { return "array[int8, 114514]" },
			`(?m)^(\s{4})resource(\s{1})`: func(string) string { return "${1}reso${2}" },
			`(?m)^(?:\s{5})fidl_union_member\[(\d+),(?:\s+)\]`: func(string) string {
				return "    yjsp_" + uuid.New().String()[:8] + " fidl_union_member[$1, array[int8, 1919810]]"
			},
		}),
		WithUnhackPatterns(map[string]func(string) string{
			`array\[int8, 114514\]`: func(string) string { return "array[]" },
			`(?m)^([ \t]*)yjsp_[a-zA-Z0-9_-]+[ \t]+fidl_union_member\[(\d+),[ \t]*array\[int8,[ \t]*1919810\]\][ \t]*$`: func(string) string {
				return "${1}fidl_union_member[$2, ]"
			},
		}),
	)
	input := "\t\tyjsp_deadbeef\tfidl_union_member[7,\tarray[int8, 1919810]]"

	got := hacker.Unhack(input)
	want := "\t\tfidl_union_member[7, ]"
	if got != want {
		t.Fatalf("Unhack() mismatch:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestHackEmptyStruct(t *testing.T) {
	hacker := NewHacker(
		WithHackPatterns(map[string]func(string) string{
			`(?m)(\w+) \{[\n ]+\}`: func(string) string { return "$1 {\n\tyjsp\tvoid\n}" },
		}),
		WithUnhackPatterns(map[string]func(string) string{
			`(?m)(\w+) \{\n\tyjsp\tvoid\n\}`: func(string) string { return "$1 {\n\t}" },
		}),
	)
	emptyStruct := `
fuchsia_ui_app_ViewProviderCreateView2RequestInLine {
     
} [packed]
`
	want := `
fuchsia_ui_app_ViewProviderCreateView2RequestInLine {
	yjsp	void
} [packed]
`
	got := hacker.Hack(emptyStruct)
	if got != want {
		t.Fatalf("hack empty struct failed: got=%v, want=%v", got, want)
	}
}
