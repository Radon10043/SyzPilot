package tools

import (
	"context"

	"github.com/Radon10043/cloud/src/generator/database"
)

// tool: get code of the global variable by its name
type GetGlobalVarCodeByName struct {
	db *database.Database
}

func NewGetGlobalVarCodeByNameTool(db *database.Database) *GetGlobalVarCodeByName {
	return &GetGlobalVarCodeByName{db: db}
}

func (t GetGlobalVarCodeByName) Name() string { return "get_global_var_code_by_name" }
func (t GetGlobalVarCodeByName) Description() string {
	return "Retrieve the code of a global variable given its name."
}
func (t GetGlobalVarCodeByName) Call(ctx context.Context, input string) (string, error) {
	globalVarRecord, err := t.db.GetGlobalVar(input)
	if err != nil {
		return "", err
	}
	return globalVarRecord.Code, nil
}
