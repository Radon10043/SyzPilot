package ast_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/ast"
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

func TestJson2syzlang(t *testing.T) {
	jpath := filepath.Join(root, "data", "test", "dev_md_gen.json")
	jbyte, err := os.ReadFile(jpath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}
	jstr := string(jbyte)
	sspec, err := ast.Json2syzlang(jstr)
	if err != nil {
		t.Fatalf("Json2syzlang failed: %v", err)
	}
	t.Logf("Syzlang Specification:\n%s", sspec)
}

func TestSyzlang2json(t *testing.T) {
	fp := filepath.Join(root, "data", "test", "all.txt")
	b, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Failed to read syzlang spec file: %v", err)
	}
	spec := string(b)
	jstr, err := ast.Syzlang2json(spec)
	if err != nil {
		t.Fatalf("Syzlang2json failed: %v", err)
	}
	t.Logf("Converted JSON Specification:\n%s", jstr)
}
