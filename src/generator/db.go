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

// connect to the SQLite database
func (db *Database) Connect() error {
	gormDB, err := gorm.Open(sqlite.Open(db.Path), &gorm.Config{})
	db.gormDB = gormDB
	return err
}

// get function code by function name
func (db *Database) GetFunctionCode(funcName string) (string, error) {
	var fun Function
	db.gormDB.Where("name = ?", funcName).First(&fun)
	if fun.Code == "" {
		return "", gorm.ErrRecordNotFound
	}
	return fun.Code, nil
}

// close the database connection
func (db *Database) Close() error {
	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
