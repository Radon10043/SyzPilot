package database

import "gorm.io/gorm"

type ConstDecl struct {
	Id    int    `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"uniqueIndex;not null"`
	Type  string
	Value string
}

type BitsDecl struct {
	Id      int `gorm:"primaryKey;autoIncrement"`
	Name    string
	Type    string
	Members string
}

type EnumDecl struct {
	Id      int    `gorm:"primaryKey;autoIncrement"`
	Name    string `gorm:"uniqueIndex;not null"`
	Type    string
	Members string
}

type ProtocolDecl struct {
	Id      int    `gorm:"primaryKey;autoIncrement"`
	Name    string `gorm:"uniqueIndex;not null"`
	Attrs   string
	Methods string
}

type ServiceDecl struct {
	Id      int    `gorm:"primaryKey;autoIncrement"`
	Name    string `gorm:"uniqueIndex;not null"`
	Members string
}

type StructDecl struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"uniqueIndex;not null"`
	Members   string
	TypeShape string
}

type TableDecl struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"uniqueIndex;not null"`
	Members   string
	TypeShape string
}

type UnionDecl struct {
	Id        int    `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"uniqueIndex;not null"`
	Members   string
	TypeShape string
}

type AliasDecl struct {
	Id   int    `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"uniqueIndex;not null"`
	Type string
}

type Declaration struct {
	Id   int    `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"uniqueIndex;not null"`
	Type string
}

// GetConstDeclByName gets the constant declaration by its name
func (db *Database) GetConstDecl(name string) (ConstDecl, error) {
	var rec ConstDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return ConstDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetBitsDecl gets the bits declaration by its name
func (db *Database) GetBitsDecl(name string) (BitsDecl, error) {
	var rec BitsDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return BitsDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetEnumDecl gets the enum declaration by its name
func (db *Database) GetEnumDecl(name string) (EnumDecl, error) {
	var rec EnumDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return EnumDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

func (pd ProtocolDecl) GetName() string { return pd.Name }
func (pd ProtocolDecl) GetCode() string { return pd.Methods }
func (pd ProtocolDecl) GetFile() string { return "" }
func (pd ProtocolDecl) GetLine() int    { return -1 }

// GetProtocolDecl gets the protocol declaration by its name
func (db *Database) GetProtocolDecl(name string) (ProtocolDecl, error) {
	var rec ProtocolDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return ProtocolDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetServiceDecl gets the service declaration by its name
func (db *Database) GetServiceDecl(name string) (ServiceDecl, error) {
	var rec ServiceDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return ServiceDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetStructDecl gets the struct declaration by its name
func (db *Database) GetStructDecl(name string) (StructDecl, error) {
	var rec StructDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return StructDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetTableDecl gets the table declaration by its name
func (db *Database) GetTableDecl(name string) (TableDecl, error) {
	var rec TableDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return TableDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetUnionDecl gets the union declaration by its name
func (db *Database) GetUnionDecl(name string) (UnionDecl, error) {
	var rec UnionDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return UnionDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetAliasDecl gets the alias declaration by its name
func (db *Database) GetAliasDecl(name string) (AliasDecl, error) {
	var rec AliasDecl
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return AliasDecl{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// GetDecl gets the declaration by its name
func (db *Database) GetDecl(name string) (Declaration, error) {
	var rec Declaration
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return Declaration{}, gorm.ErrRecordNotFound
	}
	return rec, nil
}
