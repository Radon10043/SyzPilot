package agent_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/agent"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

var testAgent agent.Agent

// init reads the .env file and initializes the testAgent
func init() {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	envFile := filepath.Join(root, ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		panic(err)
	}
	llm, err := openai.New(
		openai.WithBaseURL(os.Getenv("OPENAI_BASE_URL")),
		openai.WithToken(os.Getenv("OPENAI_API_KEY")),
		openai.WithModel("gemini-3-pro-preview"),
	)
	toolMap := make(map[string]myTools.ToolExec)
	toolMap[myTools.Toys[0].Function.Name] = myTools.ToolExec{
		Tool: myTools.Toys[0],
		Exec: myTools.ExecGetCurrentWeather,
	}
	testAgent = agent.Agent{
		Ctx:        context.Background(),
		Model:      llm,
		Messages:   []llms.MessageContent{},
		ToolMap:    toolMap,
		ToolHelper: nil,
	}
}

func TestExecWeatherTool(t *testing.T) {
	testAgent.AddHumanMessage("What is the weather like in Boston?")
	_, err := testAgent.Query()
	if err != nil {
		t.Fatalf("failed to query agent: %v", err)
	}
	err = testAgent.ExecTools()
	if err != nil {
		t.Fatalf("failed to execute tools: %v", err)
	}
	response, err := testAgent.Query()
	if err != nil {
		t.Fatalf("failed to query agent after tool execution: %v", err)
	}
	t.Logf("Response of AI: %s\n", response.Choices[0].Content)
}
