package pool_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Radon10043/cloud/src/pkg/pool"
)

var (
	ROOT  string
	SPOOL = &pool.SpecPool{}
)

func init() {
	// load env file
	_, file, _, _ := runtime.Caller(0)
	ROOT = filepath.Join(filepath.Dir(file), "..", "..", "..")

	// an example of SpecPool
	SPOOL.Insert(pool.SpecElement{
		Name:  "linux/ioctl.h",
		Type:  "include",
		Code:  "include <linux/ioctl.h>",
		Valid: true,
	})
	SPOOL.Insert(pool.SpecElement{
		Name:  "openat$xxx",
		Type:  "init_syscall",
		Code:  "openat$xxx(...)",
		Valid: true,
	})
	SPOOL.Insert(pool.SpecElement{
		Name:  "ioctl$xxx",
		Type:  "syscall",
		Code:  "ioctl$xxx(...)",
		Valid: true,
	})
	SPOOL.Insert(pool.SpecElement{
		Name:  "ioctl$yyy",
		Type:  "syscall",
		Code:  "ioctl$yyy(...)",
		Valid: true,
	})
	SPOOL.Insert(pool.SpecElement{
		Name:  "mystruct",
		Type:  "struct",
		Code:  "struct mystruct {...}",
		Valid: false,
	})
}

func TestSpecPool2Json(t *testing.T) {
	jstr, err := SPOOL.Json()
	if err != nil {
		t.Fatalf("Failed to convert SpecPool to JSON: %v", err)
	}
	_, err = pool.NewSpecPoolFromJson(jstr)
	if err != nil {
		t.Fatalf("Failed to convert JSON to SpecPool: %v", err)
	}
}

func TestSyzlang2SpecPool(t *testing.T) {
	fp := filepath.Join(ROOT, "data", "test", "all.txt")
	b, err := os.ReadFile(fp)
	if err != nil {
		t.Fatalf("Failed to read syzlang spec file: %v", err)
	}
	spec := string(b)
	spool, err := pool.NewSpecPoolFromSyzlang(spec)
	if err != nil {
		t.Fatalf("Syzlang2SpecPool failed: %v", err)
	}
	if spool.Len() != 131 {
		t.Fatalf("Expected 131 spec elements, got %d", spool.Len())
	}
}
