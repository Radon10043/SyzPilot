package agent_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/agent"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
)

var testAgent agent.Agent

// init function
func init() {
	// load env file
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	envFile := filepath.Join(root, ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		panic(err)
	}

	// init agent
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel("gpt-5-nano"),
	)
	testAgent = agent.Agent{
		Context: context.Background(),
		Model:   llm,
	}

}

func TestQuery(t *testing.T) {
	response, err := testAgent.Query("Hello, how are you?")
	if err != nil {
		t.Fatalf("Agent query failed: %v", err)
	}
	if response == "" {
		t.Error("Agent response is empty")
	}
	fmt.Println("Agent response: ", response)
}
