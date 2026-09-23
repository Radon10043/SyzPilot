package tools_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Radon10043/SyzPilot/src/pkg/agent"
	"github.com/Radon10043/SyzPilot/src/pkg/database"
	myTools "github.com/Radon10043/SyzPilot/src/pkg/tools"
	"github.com/tmc/langchaingo/llms"
)

func TestGetGlobalVarCodeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "linux.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	toolMap := map[string]myTools.ToolExec{
		myTools.GetGlobalVarCodeByNameTool.Function.Name: {
			Tool: myTools.GetGlobalVarCodeByNameTool,
			Exec: myTools.ExecGetGlobalVarCodeByName,
		},
	}
	toolHelper := &myTools.ToolHelper{
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
	myAgent.AddHumanMessage("What's the code of global variable dvb_frontend_fops?")
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
