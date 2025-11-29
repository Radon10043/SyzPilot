package tools

import (
	"context"

	"github.com/Radon10043/cloud/src/generator/database"
)

// tool: get code of the enum by its name
type GetEnumCodeByName struct {
	db *database.Database
}

func NewGetEnumCodeByNameTool(db *database.Database) *GetEnumCodeByName {
	return &GetEnumCodeByName{db: db}
}

func (t GetEnumCodeByName) Name() string { return "get_enum_code_by_name" }
func (t GetEnumCodeByName) Description() string {
	return "Retrieve the code of an enum given its name."
}
func (t GetEnumCodeByName) Call(ctx context.Context, input string) (string, error) {
	enumRecord, err := t.db.GetEnum(input)
	if err != nil {
		return "", err
	}
	return enumRecord.Code, nil
}
