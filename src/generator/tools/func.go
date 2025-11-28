package tools

import (
	"context"

	"github.com/Radon10043/cloud/src/generator/database"
)

type GetFuncCodeByName struct {
	db *database.Database
}

func NewGetFuncCodeByNameTool(db *database.Database) *GetFuncCodeByName {
	return &GetFuncCodeByName{db: db}
}

func (t GetFuncCodeByName) Name() string { return "get_func_code_by_name" }
func (t GetFuncCodeByName) Description() string {
	return "Retrieve the code of a function given its name."
}
func (t GetFuncCodeByName) Call(ctx context.Context, input string) (string, error) {
	funcRecord, err := t.db.GetFunction(input)
	if err != nil {
		return "", err
	}
	return funcRecord.Code, nil
}
