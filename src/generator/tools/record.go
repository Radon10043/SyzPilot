package tools

import (
	"context"

	"github.com/Radon10043/cloud/src/generator/database"
)

// tool: get code of the struct by its name
type GetStructCodeByName struct {
	db *database.Database
}

func NewGetStructCodeByNameTool(db *database.Database) *GetStructCodeByName {
	return &GetStructCodeByName{db: db}
}

func (t GetStructCodeByName) Name() string { return "get_struct_code_by_name" }
func (t GetStructCodeByName) Description() string {
	return "Retrieve the code of a struct given its name."
}
func (t GetStructCodeByName) Call(ctx context.Context, input string) (string, error) {
	record, err := t.db.GetStruct(input)
	if err != nil {
		return "", err
	}
	return record.Code, nil
}

// tool: get code of the union by its name
type GetUnionCodeByName struct {
	db *database.Database
}

func NewGetUnionCodeByNameTool(db *database.Database) *GetUnionCodeByName {
	return &GetUnionCodeByName{db: db}
}

func (t GetUnionCodeByName) Name() string { return "get_union_code_by_name" }
func (t GetUnionCodeByName) Description() string {
	return "Retrieve the code of a union given its name."
}
func (t GetUnionCodeByName) Call(ctx context.Context, input string) (string, error) {
	record, err := t.db.GetUnion(input)
	if err != nil {
		return "", err
	}
	return record.Code, nil
}
