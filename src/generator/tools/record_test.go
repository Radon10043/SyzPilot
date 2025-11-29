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

func TestGetStructCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "kernel.db")}
	db.Connect()
	defer func() {
		_ = db.Close()
	}()
	structTools := []tools.Tool{
		myTools.NewGetStructCodeByNameTool(&db),
	}
	ctx := context.Background()
	agent := agents.NewOpenAIFunctionsAgent(llm, structTools)
	executor := agents.NewExecutor(agent)
	res, err := chains.Call(ctx, executor, map[string]interface{}{
		"input": "What's the code of struct sound_unit?",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res["output"])
}

func TestGetUnionCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "kernel.db")}
	db.Connect()
	defer func() {
		_ = db.Close()
	}()
	unionTools := []tools.Tool{
		myTools.NewGetUnionCodeByNameTool(&db),
	}
	ctx := context.Background()
	agent := agents.NewOpenAIFunctionsAgent(llm, unionTools)
	executor := agents.NewExecutor(agent)
	res, err := chains.Call(ctx, executor, map[string]interface{}{
		"input": "What's the code of union ipvs_sockaddr?",
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res["output"])
}
