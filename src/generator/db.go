package generator

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

// close the database connection
func (db *Database) Close() error {
	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
