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
}

var ToolExecutor = map[string]func(tc llms.ToolCall) (llms.MessageContent, error){
	// "get_current_weather":               ExecGetCurrentWeather,
	"get_func_code_by_name":       ExecGetFuncCodeByName,
	"get_enum_code_by_enumerator": ExecGetEnumCodeByEnumerator,
	"get_enum_code_by_specifier":  ExecGetEnumCodeBySpecifier,
	"get_struct_code_by_name":     ExecGetStructCodeByName,
	"get_union_code_by_name":      ExecGetUnionCodeByName,
	"get_global_var_code_by_name": ExecGetGlobalVarCodeByName,
	"get_typedef_code_by_define":  ExecGetTypedefCodeByDefine,
	"get_typedef_type_by_define":  ExecGetTypedefTypeByDefine,
	"get_macro_def_code_by_name":  ExecGetMacroDefCodeByName,
}
