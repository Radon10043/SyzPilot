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
	sc   *check.SpecCheck
)

func init() {
	_, file, _, _ := runtime.Caller(0)
	root = filepath.Join(filepath.Dir(file), "..", "..", "..")
	envFile := filepath.Join(root, ".env")
	err := godotenv.Load(envFile)
	if err != nil {
		panic(err)
	}
}

func TestCheckValid(t *testing.T) {
	// create an temp directory for workdir
	wd, err := os.MkdirTemp(os.TempDir(), "cloud-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(wd)
	sc = check.NewSpecCheck(
		check.WithSyzExtract(filepath.Join(root, "bin", "syz-extract")),
		check.WithSyzCheck(filepath.Join(root, "bin", "syz-check")),
		check.WithKernelForExtract("/vol/linux/v6.12-extract"),
		check.WithKernelForCheck("/vol/linux/v6.12-check"),
		check.WithWorkdir(wd),
		check.WithSyzkaller(filepath.Join(root, "syzkaller")),
	)
	if err = sc.SetupWorkdir(); err != nil {
		t.Fatalf("failed to setup workdir: %v", err)
	}
	src := filepath.Join(root, "data", "test", "dev_md.txt")
	spec, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	dst := filepath.Join(sc.Workdir, "sys", "linux", "spec.txt")
	err = os.WriteFile(dst, spec, 0644)
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
	// create an temp directory for workdir
	wd, err := os.MkdirTemp(os.TempDir(), "cloud-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(wd)
	sc = check.NewSpecCheck(
		check.WithSyzExtract(filepath.Join(root, "bin", "syz-extract")),
		check.WithSyzCheck(filepath.Join(root, "bin", "syz-check")),
		check.WithKernelForExtract("/vol/linux/v6.12-extract"),
		check.WithKernelForCheck("/vol/linux/v6.12-check"),
		check.WithWorkdir(wd),
		check.WithSyzkaller(filepath.Join(root, "syzkaller")),
	)
	if err = sc.SetupWorkdir(); err != nil {
		t.Fatalf("failed to setup workdir: %v", err)
	}
	src := filepath.Join(root, "data", "test", "dev_md_bad.txt")
	spec, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	dst := filepath.Join(sc.Workdir, "sys", "linux", "spec.txt")
	err = os.WriteFile(dst, spec, 0644)
	if err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}
	_, stderr, err := sc.ExtractConst()
	if err == nil {
		t.Fatal("extract should be failed, but success.")
	}
	t.Logf("stderr: %v", stderr.String())
}
