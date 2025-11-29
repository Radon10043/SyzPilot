package tools_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/Radon10043/cloud/src/generator/database"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/tools"
)

func TestGetEnumCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "kernel.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	enumTools := []tools.Tool{
		myTools.NewGetEnumCodeByNameTool(&db),
	}
	ctx := context.Background()
	agent := agents.NewOpenAIFunctionsAgent(llm, enumTools)
	executor := agents.NewExecutor(agent)
	res, err := chains.Call(ctx, executor, map[string]interface{}{
		"input": "What's the code of enum bbr_mode?",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res["output"])
}
