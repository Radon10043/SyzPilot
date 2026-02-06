package utils_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/utils"
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

func TestExtractFirstCodeBlock(t *testing.T) {
	fp := filepath.Join(root, "data", "test", "dev_v4l2_gen.msg")
	mdBytes, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Failed to read Markdown file: %v", err)
	}
	mdContent := string(mdBytes)
	codeBlock, found := utils.ExtractFirstCodeBlock(mdContent, "json")
	if !found {
		t.Fatalf("No JSON code block found in the Markdown content")
	}
	t.Logf("Extracted JSON Code Block:\n%s", codeBlock)
}
