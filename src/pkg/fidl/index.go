package fidl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// entry is a single declaration in the index, used for search and lookup.
type entry struct {
	fidlName string // fully-qualified FIDL name, e.g. "fuchsia.foo/Bar"
	mangled  string // syzlang-style name, e.g. "fuchsia_foo_Bar"
	kind     string // const / enum / bits / struct / union / table / protocol / alias / new-type
	rendered string // human-readable rendering of the declaration
}

// Index holds all FIDL declarations parsed from a directory of *.fidl.json files.
type Index struct {
	byName  map[string]*entry // keyed by both fidlName and mangled name
	entries []*entry          // all declarations, sorted by mangled name
}

// Load parses every file matching pattern (a filepath.Glob pattern such as
// "<dir>/*/*.fidl.json", matching the layout of out/x64/fidling/gen/sdk/fidl) and
// builds an index of all declarations. It returns an error only if the pattern is
// malformed; files that fail to read or parse are skipped so a single bad file
// does not abort loading.
func Load(pattern string) (*Index, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob fidl json from %v: %v", pattern, err)
	}
	idx := &Index{byName: make(map[string]*entry)}
	for _, path := range matches {
		buf, err := os.ReadFile(path)
		if err != nil {
			continue // skip unreadable file
		}
		var r root
		if err := json.Unmarshal(buf, &r); err != nil {
			continue // skip malformed file
		}
		idx.addRoot(&r)
	}
	sort.Slice(idx.entries, func(i, j int) bool {
		return idx.entries[i].mangled < idx.entries[j].mangled
	})
	return idx, nil
}

// Len returns the number of declarations in the index.
func (idx *Index) Len() int {
	if idx == nil {
		return 0
	}
	return len(idx.entries)
}

func (idx *Index) addRoot(r *root) {
	for _, d := range r.ConstDeclarations {
		idx.add(d.Name, "const", d.render())
	}
	for _, d := range r.EnumDeclarations {
		idx.add(d.Name, "enum", d.render())
	}
	for _, d := range r.BitsDeclarations {
		idx.add(d.Name, "bits", d.render())
	}
	for _, d := range r.StructDeclarations {
		idx.add(d.Name, "struct", d.render("struct"))
	}
	for _, d := range r.ExternalStructDeclarations {
		idx.add(d.Name, "struct", d.render("struct"))
	}
	for _, d := range r.UnionDeclarations {
		idx.add(d.Name, "union", d.render("union"))
	}
	for _, d := range r.TableDeclarations {
		idx.add(d.Name, "table", d.render("table"))
	}
	for _, d := range r.ProtocolDeclarations {
		idx.add(d.Name, "protocol", d.render())
	}
	for _, d := range r.AliasDeclarations {
		idx.add(d.Name, "alias", d.render("alias"))
	}
	for _, d := range r.NewTypeDeclarations {
		idx.add(d.Name, "new-type", d.render("new-type"))
	}
}

func (idx *Index) add(fidlName, kind, rendered string) {
	if fidlName == "" {
		return
	}
	if _, ok := idx.byName[fidlName]; ok {
		return // first declaration wins (handles duplicate external structs)
	}
	e := &entry{
		fidlName: fidlName,
		mangled:  mangle(fidlName),
		kind:     kind,
		rendered: rendered,
	}
	idx.entries = append(idx.entries, e)
	idx.byName[e.fidlName] = e
	idx.byName[e.mangled] = e
}

// Get resolves a declaration name and returns its rendering. The query may be a
// FIDL name ("fuchsia.foo/Bar"), a syzlang-mangled name ("fuchsia_foo_Bar"), or a
// syzlang element name carrying a generated suffix ("...RequestInLine"). Returns
// false if no declaration matches.
func (idx *Index) Get(query string) (string, bool) {
	if idx == nil {
		return "", false
	}
	for _, cand := range candidates(query) {
		if e, ok := idx.byName[cand]; ok {
			return e.rendered, true
		}
	}
	return "", false
}

// Search returns up to limit declarations whose FIDL or mangled name contains
// keyword (case-insensitive), each formatted as "mangled_name (kind)".
func (idx *Index) Search(keyword string, limit int) []string {
	if idx == nil {
		return nil
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	var res []string
	for _, e := range idx.entries {
		if strings.Contains(strings.ToLower(e.mangled), keyword) ||
			strings.Contains(strings.ToLower(e.fidlName), keyword) {
			res = append(res, fmt.Sprintf("%s (%s)", e.mangled, e.kind))
			if limit > 0 && len(res) >= limit {
				break
			}
		}
	}
	return res
}

// mangle converts a FIDL name to its syzlang form by replacing "." and "/" with "_".
func mangle(fidlName string) string {
	return strings.NewReplacer(".", "_", "/", "_").Replace(fidlName)
}

// candidates returns the lookup keys to try for a query, in priority order: the
// query as-is, its mangled form, and the mangled form with generated syzlang
// suffixes (InLine/OutOfLine/Handles) stripped to recover the base declaration.
func candidates(query string) []string {
	query = strings.TrimSpace(query)
	base := mangle(query)
	cands := []string{query, base}
	for _, suf := range []string{"InLine", "OutOfLine", "Handles"} {
		if strings.HasSuffix(base, suf) {
			cands = append(cands, strings.TrimSuffix(base, suf))
		}
	}
	return cands
}
