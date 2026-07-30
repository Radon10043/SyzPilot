package csrc_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/Radon10043/cloud/src/pkg/tools/csrc"
	"github.com/tmc/langchaingo/llms"
)

func TestGetTypedefCodeByDefine(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	toolMap := map[string]tools.ToolExec{
		csrc.GetTypedefCodeByDefineTool.Function.Name: {
			Tool: csrc.GetTypedefCodeByDefineTool,
			Exec: csrc.ExecGetTypedefCodeByDefine,
		},
	}
	toolHelper := &tools.ToolHelper{
		Db: &db,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:        ctx,
		Model:      llm,
		Messages:   []llms.MessageContent{},
		ToolHelper: toolHelper,
		ToolMap:    toolMap,
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
	defer db.Close()
	toolMap := map[string]tools.ToolExec{
		csrc.GetTypedefTypeByDefineTool.Function.Name: {
			Tool: csrc.GetTypedefTypeByDefineTool,
			Exec: csrc.ExecGetTypedefTypeByDefine,
		},
	}
	toolHelper := &tools.ToolHelper{
		Db: &db,
	}
	ctx := context.Background()
	myAgent := agent.Agent{
		Ctx:        ctx,
		Model:      llm,
		Messages:   []llms.MessageContent{},
		ToolHelper: toolHelper,
		ToolMap:    toolMap,
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
