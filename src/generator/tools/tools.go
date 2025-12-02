package tools

import (
	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/Radon10043/cloud/src/generator/database"
	"github.com/tmc/langchaingo/llms"
)

var DB *database.Database
var SC *check.SyzCheck

var ToolList = []llms.Tool{
	GetFuncCodeByNameTool,
	GetEnumCodeByNameTool,
	GetGlobalVarCodeByNameTool,
	GetStructCodeByNameTool,
	GetUnionCodeByNameTool,
	GetTypedefCodeByDefineTool,
	GetTypedefTypeByDefineTool,
	CheckSpecValidityTool,
}
