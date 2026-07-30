// remove unused elements under a specific directory inplace

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/google/syzkaller/pkg/ast"
	"github.com/google/syzkaller/pkg/compiler"
	"github.com/google/syzkaller/sys/targets"
)

var (
	flagIndir string
)

func main() {
	flag.StringVar(&flagIndir, "indir", "", "directory containing syzlang specs to remove unused elements.")
	flag.Parse()

	indir, err := filepath.Abs(flagIndir)
	if err != nil {
		panic(err)
	}

	// collect unused elements
	specPattern := filepath.Join(indir, "*.txt")
	desc := ast.ParseGlob(specPattern, func(pos ast.Pos, msg string) {
		panic(fmt.Sprintf("failed to parse syzlang spec %v: %v", pos, msg))
	})
	osName := filepath.Base(indir)
	target := targets.Get(osName, runtime.GOARCH)
	unusedNodes, err := compiler.CollectUnused(desc, target, func(pos ast.Pos, msg string) {
		panic(fmt.Sprintf("failed to collect unused elements %v: %v", pos, msg))
	})
	if err != nil {
		panic(err)
	}

	// collect involved files and rewrite specs to exclude unused nodes
	involveFiles := make(map[string]bool)
	for _, node := range unusedNodes {
		pos, _, _ := node.Info()
		involveFiles[pos.File] = true
	}
	rewriteMap := make(map[string][]ast.Node)
	for _, node := range desc.Nodes {
		pos, _, _ := node.Info()
		if _, ok := involveFiles[pos.File]; !ok {
			continue
		}
		if slices.Contains(unusedNodes, node) {
			continue
		}
		if isTypedefExpand(desc, node) {
			continue
		}
		rewriteMap[pos.File] = append(rewriteMap[pos.File], node)
	}

	for file, nodes := range rewriteMap {
		nDesc := &ast.Description{
			Nodes: nodes,
		}
		fileinfo, err := os.Stat(file)
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile(file, ast.Format(nDesc), fileinfo.Mode()); err != nil {
			panic(err)
		}
	}

	if len(rewriteMap) > 0 {
		fmt.Printf("Done, rewrite following files:\n")
		for file := range rewriteMap {
			fmt.Printf("  %s\n", file)
		}
	} else {
		fmt.Printf("No unused elements found.\n")
	}
}

// isTypedefExpand checks if the node is a typedef expansion
func isTypedefExpand(desc *ast.Description, node ast.Node) bool {
	if _, ok := node.(*ast.TypeDef); ok {
		return false
	}
	typedefs := make(map[string]bool)
	for _, node := range desc.Nodes {
		if _, ok := node.(*ast.TypeDef); ok {
			_, _, name := node.Info()
			typedefs[name] = true
		}
	}
	_, _, name := node.Info()
	snip := strings.Split(name, "[")
	if len(snip) < 2 {
		return false
	}
	return typedefs[snip[0]]
}
