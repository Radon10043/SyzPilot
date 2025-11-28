package tools_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/database"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
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
		openai.WithModel("gpt-5-nano"),
	)
	if err != nil {
		panic(err)
	}
}

func TestGetFuncCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "kernel.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	funcTools := []tools.Tool{
		myTools.NewGetFuncCodeByNameTool(&db),
	}
	ctx := context.Background()
	agent := agents.NewOpenAIFunctionsAgent(llm, funcTools)
	executor := agents.NewExecutor(agent)
	res, err := chains.Call(ctx, executor, map[string]interface{}{
		"input": "What's the code of pppox_ioctl function?",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res["output"])
}
