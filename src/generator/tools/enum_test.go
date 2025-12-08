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

func TestGetEnumCodeByEnumerator(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	enumTools := []llms.Tool{
		myTools.GetEnumCodeByEnumeratorTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    enumTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the enum code of enumerator DM_VERSION_CMD?")
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

func TestGetEnumCodeBySpecifier(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	enumTools := []llms.Tool{
		myTools.GetEnumCodeBySpecifierTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    enumTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the enum code of specifier bbr_mode?")
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
