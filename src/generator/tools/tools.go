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
	GetEnumCodeByEnumeratorTool,
	GetEnumCodeBySpecifierTool,
	GetGlobalVarCodeByNameTool,
	GetStructCodeByNameTool,
	GetUnionCodeByNameTool,
	GetTypedefCodeByDefineTool,
	GetTypedefTypeByDefineTool,
	GetMacroDefCodeByNameTool,
	GetMacroDefCodesByPatternTool,
	GetMacroDefLocByNameTool,
}

var ToolExecutor = map[string]func(tc llms.ToolCall) (llms.MessageContent, error){
	// Toys[0].Function.Name:                       ExecGetCurrentWeather,
	GetFuncCodeByNameTool.Function.Name:         ExecGetFuncCodeByName,
	GetEnumCodeByEnumeratorTool.Function.Name:   ExecGetEnumCodeByEnumerator,
	GetEnumCodeBySpecifierTool.Function.Name:    ExecGetEnumCodeBySpecifier,
	GetStructCodeByNameTool.Function.Name:       ExecGetStructCodeByName,
	GetUnionCodeByNameTool.Function.Name:        ExecGetUnionCodeByName,
	GetGlobalVarCodeByNameTool.Function.Name:    ExecGetGlobalVarCodeByName,
	GetTypedefCodeByDefineTool.Function.Name:    ExecGetTypedefCodeByDefine,
	GetTypedefTypeByDefineTool.Function.Name:    ExecGetTypedefTypeByDefine,
	GetMacroDefCodeByNameTool.Function.Name:     ExecGetMacroDefCodeByName,
	GetMacroDefCodesByPatternTool.Function.Name: ExecGetMacroDefCodesByPattern,
	GetMacroDefLocByNameTool.Function.Name:      ExecGetMacroDefLocByName,
}
