package fuchsia_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/agent"
	"github.com/Radon10043/cloud/src/pkg/database"
	myTools "github.com/Radon10043/cloud/src/pkg/tools"
	fuchsiaTools "github.com/Radon10043/cloud/src/pkg/tools/fuchsia"
	"github.com/tmc/langchaingo/llms"
)

func TestGetStructMembersByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "fuchsia.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	toolMap := map[string]myTools.ToolExec{
		fuchsiaTools.GetStructMembersByNameTool.Function.Name: {
			Tool: fuchsiaTools.GetStructMembersByNameTool,
			Exec: fuchsiaTools.ExecGetStructMembersByName,
		},
	}
	toolHelper := &myTools.ToolHelper{
		Db: &db,
	}
	ctx := context.Background()
	a := agent.Agent{
		Ctx:        ctx,
		Model:      llm,
		Messages:   []llms.MessageContent{},
		ToolMap:    toolMap,
		ToolHelper: toolHelper,
	}
	a.AddHumanMessage("What members are included in struct fuchsia.accessibility/ColorTransformHandlerSetColorTransformConfigurationRequest?")
	_, err = a.Query()
	if err != nil {
		t.Fatal(err)
	}
	err = a.ExecTools()
	if err != nil {
		t.Fatal(err)
	}
	response, err := a.Query()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Response of agent: %v\n", response.Choices[0].Content)
}

func TestGetStructTypeShapeByName(t *testing.T) {
	db := database.Database{Path: filepath.Join(root, "data", "database", "fuchsia.db")}
	err := db.Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	toolMap := map[string]myTools.ToolExec{
		fuchsiaTools.GetStructTypeShapeByNameTool.Function.Name: {
			Tool: fuchsiaTools.GetStructTypeShapeByNameTool,
			Exec: fuchsiaTools.ExecGetStructTypeShapeByName,
		},
	}
	toolHelper := &myTools.ToolHelper{
		Db: &db,
	}
	ctx := context.Background()
	a := agent.Agent{
		Ctx:        ctx,
		Model:      llm,
		Messages:   []llms.MessageContent{},
		ToolMap:    toolMap,
		ToolHelper: toolHelper,
	}
	a.AddHumanMessage("What is the type shape of struct fuchsia.accessibility/ColorTransformHandlerSetColorTransformConfigurationRequest?")
	_, err = a.Query()
	if err != nil {
		t.Fatal(err)
	}
	err = a.ExecTools()
	if err != nil {
		t.Fatal(err)
	}
	response, err := a.Query()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Response of agent: %v\n", response.Choices[0].Content)
}
