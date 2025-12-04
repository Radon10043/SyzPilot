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

func TestGetStructCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	db.Connect()
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	structTools := []llms.Tool{
		myTools.GetStructCodeByNameTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    structTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the code of struct sound_unit?")
	_, err := myAgent.Query()
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

func TestGetUnionCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	db.Connect()
	defer func() {
		_ = db.Close()
	}()
	myTools.DB = &db
	unionTools := []llms.Tool{
		myTools.GetUnionCodeByNameTool,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:      ctx,
		Model:    llm,
		Tools:    unionTools,
		Messages: []llms.MessageContent{},
	}
	myAgent.AddHumanMessage("What's the code of union ipvs_sockaddr?")
	_, err := myAgent.Query()
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
