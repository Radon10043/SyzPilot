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

func TestGetEnumCodeByEnumerator(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	toolMap := map[string]tools.ToolExec{
		csrc.GetEnumCodeByEnumeratorTool.Function.Name: {
			Tool: csrc.GetEnumCodeByEnumeratorTool,
			Exec: csrc.ExecGetEnumCodeByEnumerator,
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
		ToolMap:    toolMap,
		ToolHelper: toolHelper,
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
	defer db.Close()
	toolMap := map[string]tools.ToolExec{
		csrc.GetEnumCodeBySpecifierTool.Function.Name: {
			Tool: csrc.GetEnumCodeBySpecifierTool,
			Exec: csrc.ExecGetEnumCodeBySpecifier,
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
		ToolMap:    toolMap,
		ToolHelper: toolHelper,
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
