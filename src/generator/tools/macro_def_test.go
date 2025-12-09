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

func TestGetMacroDefCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	myTools.DB = &db
	macroDefTools := []llms.Tool{
		myTools.GetMacroDefCodeByNameTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    macroDefTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the code of macro definition EXT4_EPOCH_BITS?")
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

func TestGetMacroDefCodesByPrefix(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	myTools.DB = &db
	macroDefTools := []llms.Tool{
		myTools.GetMacroDefCodesByPrefixTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    macroDefTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What are the codes of macro definitions starting with 'MEDIA_PAD_FL_'?")
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
