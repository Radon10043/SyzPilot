package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/Radon10043/cloud/src/pkg/database"
	"github.com/Radon10043/cloud/src/pkg/fidlir"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	flagFidl  string // path to the fidl directory, e.g. fuchsia/out/x64/fidling/gen/sdk/fidl
	flagOutdb string // path to the output database
)

func main() {
	// setup and parse command line options
	curdir, err := os.Getwd()
	if err != nil {
		panic("failed to get current working directory: " + err.Error())
	}
	flag.StringVar(&flagFidl, "fidl", "", "path to the fidl directory, e.g. fuchsia/out/x64/fidling/gen/sdk/fidl")
	flag.StringVar(&flagOutdb, "outdb", filepath.Join(curdir, "fuchsia.db"), "path to the output database")
	flag.Parse()

	// convert to the abs path
	var fidlpath, dbpath string
	if fidlpath, err = filepath.Abs(flagFidl); err != nil {
		panic("failed to get abs path of fidl directory: " + err.Error())
	}
	if dbpath, err = filepath.Abs(flagOutdb); err != nil {
		panic("failed to get abs path of output database: " + err.Error())
	}

	// match *.fidl.json files via fidlpath/*/*.fidl.json
	matches, err := filepath.Glob(filepath.Join(fidlpath, "*", "*.fidl.json"))
	if err != nil {
		panic("failed to glob *.fidl.json files: " + err.Error())
	}
	if len(matches) == 0 {
		log.Fatalf("no *.fidl.json files found in %s, stop.", fidlpath)
	}

	// connect to the SQLite database
	dsn := "file:" + dbpath + "?" +
		"_pragma=busy_timeout(5000)&" +
		"_pragma=journal_mode(WAL)&" +
		"_pragma=foreign_keys(ON)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to the SQLite database: " + err.Error())
	}

	// create tables
	db.AutoMigrate(
		&database.ConstDecl{},
		&database.BitsDecl{},
		&database.EnumDecl{},
		&database.ProtocolDecl{},
		&database.ServiceDecl{},
		&database.StructDecl{},
		&database.TableDecl{},
		&database.UnionDecl{},
		&database.AliasDecl{},
		&database.Declaration{},
	)

	// analyze each fidljson file and store the result in the output database
	for i, match := range matches {
		log.Printf("[%d/%d] processing %s ...", i+1, len(matches), match)
		fidlirData, err := analyze(match)
		if err != nil {
			log.Fatalf("failed to analyze %s: %v", match, err)
		}
		normalize(fidlirData)
		if err := insertIntoDB(db, fidlirData); err != nil {
			log.Fatalf("failed to insert data into database for %s: %v", match, err)
		}
	}

	// close the database connection when done
	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get database connection: " + err.Error())
	}
	sqlDB.Close()
}

// normalize normalizes the fidlir data to ensure that all fields are properly initialized.
func normalize(fidlirData *fidlir.Fidlir) {
	for i := range fidlirData.ProtocolDecls {
		p := &fidlirData.ProtocolDecls[i]
		if p.Attrs == nil {
			p.Attrs = []fidlir.Attribute{}
		}
	}
}

// insertIntoDB inserts the fidlir data into the database. It returns error
// if failed to insert.
func insertIntoDB(db *gorm.DB, fidlirData *fidlir.Fidlir) error {
	// const declarations
	if res := db.CreateInBatches(
		fidlirData.ConstDecls, len(fidlirData.ConstDecls),
	); res.Error != nil {
		return res.Error
	}

	// bits declarations
	if res := db.CreateInBatches(
		fidlirData.BitsDecls, len(fidlirData.BitsDecls),
	); res.Error != nil {
		return res.Error
	}

	// enum declarations
	if res := db.CreateInBatches(
		fidlirData.EnumDecls, len(fidlirData.EnumDecls),
	); res.Error != nil {
		return res.Error
	}

	// protocol declarations
	if res := db.CreateInBatches(
		fidlirData.ProtocolDecls, len(fidlirData.ProtocolDecls),
	); res.Error != nil {
		return res.Error
	}

	// service declarations
	if res := db.CreateInBatches(
		fidlirData.ServiceDecls, len(fidlirData.ServiceDecls),
	); res.Error != nil {
		return res.Error
	}

	// struct declarations
	if res := db.CreateInBatches(
		fidlirData.StructDecls, len(fidlirData.StructDecls),
	); res.Error != nil {
		return res.Error
	}

	// table declarations
	if res := db.CreateInBatches(
		fidlirData.TableDecls, len(fidlirData.TableDecls),
	); res.Error != nil {
		return res.Error
	}

	// union declarations
	if res := db.CreateInBatches(
		fidlirData.UnionDecls, len(fidlirData.UnionDecls),
	); res.Error != nil {
		return res.Error
	}

	// alias declarations
	if res := db.CreateInBatches(
		fidlirData.AliasDecls, len(fidlirData.AliasDecls),
	); res.Error != nil {
		return res.Error
	}

	// declarations
	decls := make([]database.Declaration, 0, len(fidlirData.Declarations))
	for name, typ := range fidlirData.Declarations {
		decls = append(decls, database.Declaration{
			Name: name,
			Type: typ,
		})
	}
	if res := db.CreateInBatches(decls, len(decls)); res.Error != nil {
		return res.Error
	}

	return nil
}

// analyze analyzes a single fidljson file. Returns fidlir data and error message.
func analyze(fidljson string) (*fidlir.Fidlir, error) {
	b, err := os.ReadFile(fidljson)
	if err != nil {
		return nil, err
	}

	var data fidlir.Fidlir
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
