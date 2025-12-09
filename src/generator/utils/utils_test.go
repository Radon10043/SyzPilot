package utils_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/utils"
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
	jpath := filepath.Join(root, "data", "test", "dev_cec_gen.json")
	jbyte, err := os.ReadFile(jpath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}
	jstr := string(jbyte)
	sspec, err := utils.Json2syzlang(jstr)
	if err != nil {
		t.Fatalf("Json2syzlang failed: %v", err)
	}
	t.Logf("Syzlang Specification:\n%s", sspec)
}
