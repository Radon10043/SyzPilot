package tools_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Radon10043/cloud/src/generator/agent"
	"github.com/Radon10043/cloud/src/generator/database"
	myTools "github.com/Radon10043/cloud/src/generator/tools"
	"github.com/tmc/langchaingo/llms"
)

func TestGetFuncCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	funcTools := []llms.Tool{
		myTools.GetFuncCodeByNameTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    funcTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the code of pppox_ioctl function?")
	_, err = myAgent.Query()
	if err != nil {
		t.Fatal(err)
	}
	err = myAgent.ExecTools()
	if err != nil {
		t.Fatal(err)
	}
	response, err := myAgent.Query()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Response of agent: %v\n", response.Choices[0].Content)
}
