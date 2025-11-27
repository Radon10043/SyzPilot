package generator_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator"
)

// change working directory to project root
func ChangeToProjectRoot() {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	err := os.Chdir(root)
	if err != nil {
		panic(err)
	}
}

func TestFindFunction(t *testing.T) {
	ChangeToProjectRoot()
	db := generator.Database{Path: "data/kernel.db"}
	err := db.Connect()
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
	fun, err := db.GetFunction("pppox_ioctl")
	if err != nil {
		t.Errorf("Error retrieving function code: %v", err)
	}
	if fun.Code == "" {
		t.Error("Function code is empty")
	}
	db.Close()
}

func TestFindStruct(t *testing.T) {
	ChangeToProjectRoot()
	db := generator.Database{Path: "data/kernel.db"}
	err := db.Connect()
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
	rec, err := db.GetRecord("sound_unit")
	if err != nil {
		t.Errorf("Error retrieving record code: %v", err)
	}
	if rec.Type != "struct" {
		t.Errorf("Expected record type 'struct', got '%s'", rec.Type)
	}
	if rec.Code == "" {
		t.Error("Record code is empty")
	}
	db.Close()
}

func TestFindUnion(t *testing.T) {
	ChangeToProjectRoot()
	db := generator.Database{Path: "data/kernel.db"}
	err := db.Connect()
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
	rec, err := db.GetRecord("ipvs_sockaddr")
	if err != nil {
		t.Errorf("Error retrieving record code: %v", err)
	}
	if rec.Type != "union" {
		t.Errorf("Expected record type 'union', got '%s'", rec.Type)
	}
	if rec.Code == "" {
		t.Error("Record code is empty")
	}
	db.Close()
}
