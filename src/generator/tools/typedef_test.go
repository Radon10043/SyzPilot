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

func TestGetTypedefCodeByDefine(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	typedefTools := []llms.Tool{
		myTools.GetTypedefCodeByDefineTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    typedefTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the code of typedef unative_t?")
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

func TestGetTypedefTypeByDefine(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	typedefTools := []llms.Tool{
		myTools.GetTypedefTypeByDefineTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    typedefTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the type of typedef unative_t?")
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
