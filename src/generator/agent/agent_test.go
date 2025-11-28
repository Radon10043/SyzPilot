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
		Tools:   agent.MyTools,
	}

}

func TestQuery(t *testing.T) {
	response, err := testAgent.Query("Hello, how are you?")
	if err != nil {
		t.Fatalf("Agent query failed: %v", err)
	}
	if response == nil {
		t.Error("Agent response is empty")
	}
	fmt.Println("Agent response: ", response)
}

func TestFuncCalling(t *testing.T) {
	_, err := testAgent.Query("What is the tomorrow weather in New York?")
	if err != nil {
		t.Fatalf("Agent query failed: %v", err)
	}
	resp2, err := testAgent.ExecuteToolCalls()
	if err != nil {
		t.Fatalf("ExecuteToolCalls failed: %v", err)
	}
	testAgent.Messages = append(testAgent.Messages, resp2)
	resp3, err := testAgent.Model.GenerateContent(testAgent.Context, testAgent.Messages)
	if err != nil {
		t.Fatalf("Model GenerateContent failed: %v", err)
	}
	fmt.Println("Agent response: ", resp3.Choices[0].Content)
}
