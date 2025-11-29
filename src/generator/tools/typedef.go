package tools

import (
	"context"

	"github.com/Radon10043/cloud/src/generator/database"
)

// tool: get typedef's code by define
type GetTypedefCodeByDefine struct {
	db *database.Database
}

func NewGetTypedefCodeByDefineTool(db *database.Database) *GetTypedefCodeByDefine {
	return &GetTypedefCodeByDefine{db: db}
}

func (t GetTypedefCodeByDefine) Name() string { return "get_typedef_code_by_define" }
func (t GetTypedefCodeByDefine) Description() string {
	return "Retrieve the code of a typedef given its define."
}
func (t GetTypedefCodeByDefine) Call(ctx context.Context, input string) (string, error) {
	typedefRecord, err := t.db.GetTypedef(input)
	if err != nil {
		return "", err
	}
	return typedefRecord.Code, nil
}

type GetTypedefTypeByDefine struct {
	db *database.Database
}

// tool: get typedef's type by define
func NewGetTypedefTypeByDefineTool(db *database.Database) *GetTypedefTypeByDefine {
	return &GetTypedefTypeByDefine{db: db}
}

func (t GetTypedefTypeByDefine) Name() string { return "get_typedef_type_by_define" }
func (t GetTypedefTypeByDefine) Description() string {
	return "Retrieve the type of a typedef given its define."
}
func (t GetTypedefTypeByDefine) Call(ctx context.Context, input string) (string, error) {
	typedefRecord, err := t.db.GetTypedef(input)
	if err != nil {
		return "", err
	}
	return typedefRecord.Type, nil
}
