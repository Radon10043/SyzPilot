package ast_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/ast"
	"github.com/joho/godotenv"
)

var root string // root path of the project

// init function
func init() {
	// load env file
	_, file, _, _ := runtime.Caller(0)
	root = filepath.Join(filepath.Dir(file), "..", "..", "..")
	envFile := filepath.Join(root, ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		panic(err)
	}
}

func TestSyzlang2SyzSpecRecord(t *testing.T) {
	dir := filepath.Join(root, "syzkaller", "sys", "linux")
	ssr, err := ast.Syzlang2SyzSpecRecord(dir)
	if err != nil {
		t.Fatalf("Syzlang2SyzSpecRecord failed: %v", err)
	}
	t.Logf(
		"SyzSpecRecord:\nIncludes: %v\nResources: %v\nDefines: %v\nSyscalls: %v\nFlags: %v\nStructs: %v\nUnions: %v\nTypeAliases: %v\nTypeTemplates: %v",
		len(ssr.Include), len(ssr.Resource), len(ssr.Define), len(ssr.Syscall),
		len(ssr.Flags), len(ssr.Struct), len(ssr.Union), len(ssr.TypeAlias), len(ssr.TypeTemplate),
	)
}
