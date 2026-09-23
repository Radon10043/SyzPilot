package ast_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/SyzPilot/src/pkg/ast"
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

func TestAddSuffix(t *testing.T) {
	b, err := os.ReadFile(
		filepath.Join(root, "data", "test", "dev_md.txt"),
	)
	if err != nil {
		t.Fatalf("Failed to read syzlang spec file: %v", err)
	}
	syzl := string(b)
	suffix := "_ctl_fops"
	nsyzl, err := ast.AddSuffix(syzl, suffix)
	if err != nil {
		t.Fatalf("Failed to add suffix to syzlang spec: %v", err)
	}
	t.Logf("New syzlang spec: %v", nsyzl)
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
