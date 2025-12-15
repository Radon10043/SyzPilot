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
	wd, err := os.MkdirTemp(os.TempDir(), "cloud-*")
	if err != nil {
		panic(err)
	}
	sc = check.NewSpecCheck(
		check.WithSyzExtract(filepath.Join(root, "bin", "syz-extract")),
		check.WithSyzCheck(filepath.Join(root, "bin", "syz-check")),
		check.WithKernelForExtract("/vol/linux/v6.12-extract"),
		check.WithKernelForCheck("/vol/linux/v6.12-check"),
		check.WithWorkdir(wd),
		check.WithSyzkaller(filepath.Join(root, "syzkaller")),
	)
	if err = sc.SetupWorkdir(); err != nil {
		panic(err)
	}
}

func TestCheckValid(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_md.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	if err = sc.AddSpec(string(b)); err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst()
	if !valid {
		t.Fatalf("syz-extract report spec is invalid, exptected valid.")
	}
	_, _, valid = sc.CheckValidity()
	if err != nil {
		t.Fatalf("syz-check report spec is invalid, expected valid. ")
	}
	os.RemoveAll(sc.Workdir)
}

func TestCheckInvalid(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_md_bad.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	if err = sc.AddSpec(string(b)); err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst()
	if valid {
		t.Fatal("syz-extract report spec is valid, expected invalid.")
	}
	os.RemoveAll(sc.Workdir)
}

func TestEnableIgnRedeclErr(t *testing.T) {
	sc.IgnRedeclErr = true
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_snd_hw_redecl.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	if err = sc.AddSpec(string(b)); err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst()
	if !valid { // spec should be valid since ignore redeclare error is enabled
		t.Fatal("syz-extract report spec is invalid, expected valid.")
	}
	_, _, valid = sc.CheckValidity()
	if !valid {
		t.Fatal("syz-extract report spec is invalid, expected valid.")
	}
	os.RemoveAll(sc.Workdir)
}

func TestDisableIgnRedeclErr(t *testing.T) {
	// default is false
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_snd_hw_redecl.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	if err = sc.AddSpec(string(b)); err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst()
	if valid { // spec should be valid since ignore redeclare error is enabled
		t.Fatal("syz-extract report spec is valid, expected invalid.")
	}
	_, _, valid = sc.CheckValidity()
	if valid {
		t.Fatal("syz-extract report spec is valid, expected invalid.")
	}
	os.RemoveAll(sc.Workdir)
}
