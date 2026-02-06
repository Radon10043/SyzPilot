package tools_test

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	llm  *openai.LLM
	root string
)

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
	// init llm
	llm, err = openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel("gemini-2.5-flash"),
	)
	if err != nil {
		panic(err)
	}
}
