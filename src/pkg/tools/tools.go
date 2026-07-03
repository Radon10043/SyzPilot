package tools

import (
	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/fidl"
	"github.com/tmc/langchaingo/llms"
)

type ToolExec struct {
	Tool llms.Tool
	Exec func(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error)
}

type ToolHelper struct {
	Db   *database.Database
	Fidl *fidl.Index
}
