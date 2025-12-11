package tools

import (
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

type ToolExec struct {
	Tool llms.Tool
	Exec func(tc *llms.ToolCall, th *ToolHelper) (llms.MessageContent, error)
}

type ToolHelper struct {
	Db *database.Database
	Sc *check.SyzCheck
}
