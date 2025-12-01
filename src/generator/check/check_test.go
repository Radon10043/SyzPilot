package check_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/generator/check"
	"github.com/joho/godotenv"
)

var (
	root string
	sc   check.SyzCheck
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	root = filepath.Join(filepath.Dir(file), "..", "..", "..")
	envFile := filepath.Join(root, ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		panic(err)
	}
	sc = check.SyzCheck{
		Bin:              filepath.Join(root, "bin", "syz-check"),
		Workdir:          filepath.Join(root, "syzkaller"),
		KernelForExtract: "/vol/linux/v6.12-extract",
		KernelForCheck:   "/vol/linux/v6.12-check",
	}
}

func TestCheckValid(t *testing.T) {
	err := sc.CheckWorkdir()
	if err != nil {
		t.Fatalf("check workdir failed: %v", err)
	}
	err = sc.CleanWorkdir()
	if err != nil {
		t.Fatalf("clean workdir failed: %v", err)
	}
	srcPath := filepath.Join(root, "data", "test", "dev_md.txt")
	spec, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	dstPath := filepath.Join(sc.Workdir, "sys", "linux", "spec.txt")
	err = os.WriteFile(dstPath, spec, 0644)
	if err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	_, stderr, err := sc.ExtractConst()
	if err != nil {
		t.Fatalf("extract const failed: %v\nstderr: %v", err, stderr.String())
	}
	_, stderr, err = sc.CheckValidity()
	if err != nil {
		t.Fatalf("syz-check failed: %v\nstderr: %v", err, stderr.String())
	}
}

func TestCheckInvalid(t *testing.T) {
	err := sc.CheckWorkdir()
	if err != nil {
		t.Fatalf("check workdir failed: %v", err)
	}
	err = sc.CleanWorkdir()
	if err != nil {
		t.Fatalf("clean workdir failed: %v", err)
	}
	srcPath := filepath.Join(root, "data", "test", "dev_md_bad.txt")
	spec, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	dstPath := filepath.Join(sc.Workdir, "sys", "linux", "spec.txt")
	err = os.WriteFile(dstPath, spec, 0644)
	if err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	_, _, err = sc.ExtractConst()
	if err == nil {
		t.Fatal("`make extract` should be failed, but success.")
	}
}
