package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	Path   string   // path to the SQLite database file
	gormDB *gorm.DB // GORM database connection
}

type Function struct {
	Id   int    // unique function ID
	Name string // function name
	File string // file where the function is located
	Line int    // line number in the file
	Code string // function code
}

type Record struct {
	Id   int    // unique record ID
	Name string // record name
	Type string // record type, struct or union
	File string // file where the record is located
	Line int    // line number in the file
	Code string // record code
}

type Enum struct {
	Id   int    // unique enum ID
	Name string // enum name
	File string // file where the enum is located
	Line int    // line number in the file
	Code string // enum code
}

type Typedef struct {
	Id     int    // unique typedef ID
	Type   string // old name in typedef
	Define string // new name in typedef
	File   string // file where the typedef is located
	Line   int    // line number in the file
	Code   string // typedef code
}

type GlobalVar struct {
	Id   int    // unique global variable ID
	Name string // global variable name
	File string // file where the global variable is located
	Line int    // line number in the file
	Code string // global variable code
}

// connect to the SQLite database
func (db *Database) Connect() error {
	gormDB, err := gorm.Open(sqlite.Open(db.Path), &gorm.Config{})
	db.gormDB = gormDB
	return err
}

// get function data by function name
func (db *Database) GetFunction(name string) (Function, error) {
	var fun Function
	db.gormDB.Where("name = ?", name).First(&fun)
	if fun.Id == 0 {
		return fun, gorm.ErrRecordNotFound
	}
	return fun, nil
}

// get record data by record name
func (db *Database) GetRecord(name string) (Record, error) {
	var rec Record
	db.gormDB.Where("name = ?", name).First(&rec)
	if rec.Id == 0 {
		return rec, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// get struct data by struct name
func (db *Database) GetStruct(name string) (Record, error) {
	var rec Record
	db.gormDB.Where("name = ? AND type = ?", name, "struct").First(&rec)
	if rec.Id == 0 {
		return rec, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// get union data by union name
func (db *Database) GetUnion(name string) (Record, error) {
	var rec Record
	db.gormDB.Where("name = ? AND type = ?", name, "union").First(&rec)
	if rec.Id == 0 {
		return rec, gorm.ErrRecordNotFound
	}
	return rec, nil
}

// get enum data by enum name
func (db *Database) GetEnum(name string) (Enum, error) {
	var enum Enum
	db.gormDB.Where("name = ?", name).First(&enum)
	if enum.Id == 0 {
		return enum, gorm.ErrRecordNotFound
	}
	return enum, nil
}

// get typedef data by define (new name)
func (db *Database) GetTypedef(def string) (Typedef, error) {
	var td Typedef
	db.gormDB.Where("define = ?", def).First(&td)
	if td.Id == 0 {
		return td, gorm.ErrRecordNotFound
	}
	return td, nil
}

// specify table name for GlobalVar model
func (GlobalVar) TableName() string {
	return "globalVars"
}

// get global variable data by variable name
func (db *Database) GetGlobalVar(name string) (GlobalVar, error) {
	var gv GlobalVar
	db.gormDB.Where("name = ?", name).First(&gv)
	if gv.Id == 0 {
		return gv, gorm.ErrRecordNotFound
	}
	return gv, nil
}

// get all global variables from the database
func (db *Database) GetAllGlobalVar() ([]GlobalVar, error) {
	var gvs []GlobalVar
	result := db.gormDB.Find(&gvs)
	return gvs, result.Error
}

// close the database connection
func (db *Database) Close() error {
	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
