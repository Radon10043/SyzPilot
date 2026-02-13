package check_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/check"
	osu "github.com/Radon10043/cloud/src/pkg/osutil"
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
	osType, err := osu.ParseOsType(runtime.GOOS)
	sc = check.NewSpecCheck(
		check.WithOs(osType),
		check.WithSyzExtract(filepath.Join(root, "bin", "syz-extract")),
		check.WithSyzCheck(filepath.Join(root, "bin", "syz-check")),
		check.WithKernelForExtract("/vol/linux/v6.18-extract"),
		check.WithKernelForCheck("/vol/linux/v6.18-check"),
		check.WithSysdir(filepath.Join(root, "syzkaller", "sys")),
		check.WithWorkdir(wd),
	)
	if err = sc.SetupWorkdir(); err != nil {
		panic(err)
	}
}

func TestCheckValidFreebsd(t *testing.T) {
	if runtime.GOOS != "freebsd" {
		t.Skip("skip freebsd specific test on non-freebsd OS")
	}
	b, err := os.ReadFile(
		filepath.Join(root, "data", "test", "dev_aac.txt"),
	)
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	fpath, err := sc.AddSpec(string(b))
	if err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst(filepath.Base(fpath))
	if !valid {
		t.Fatalf("syz-extract report spec is invalid, exptected valid.")
	}
	_, _, valid = sc.CheckValidity()
	if !valid {
		t.Fatalf("syz-check report spec is invalid, expected valid. ")
	}
	os.RemoveAll(sc.Workdir)
}

func TestCheckValid(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_md.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	fpath, err := sc.AddSpec(string(b))
	if err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst(filepath.Base(fpath))
	if !valid {
		t.Fatalf("syz-extract report spec is invalid, exptected valid.")
	}
	_, _, valid = sc.CheckValidity()
	if !valid {
		t.Fatalf("syz-check report spec is invalid, expected valid. ")
	}
	os.RemoveAll(sc.Workdir)
}

func TestCheckInvalid(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "data", "test", "dev_md_bad.txt"))
	if err != nil {
		t.Fatalf("failed to read spec file: %v", err)
	}
	fpath, err := sc.AddSpec(string(b))
	if err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst(filepath.Base(fpath))
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
	fpath, err := sc.AddSpec(string(b))
	if err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst(filepath.Base(fpath))
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
	fpath, err := sc.AddSpec(string(b))
	if err != nil {
		t.Fatalf("failed to add spec: %v", err)
	}
	_, _, valid := sc.ExtractConst(filepath.Base(fpath))
	if valid { // spec should be valid since ignore redeclare error is enabled
		t.Fatal("syz-extract report spec is valid, expected invalid.")
	}
	_, _, valid = sc.CheckValidity()
	if valid {
		t.Fatal("syz-extract report spec is valid, expected invalid.")
	}
	os.RemoveAll(sc.Workdir)
}
