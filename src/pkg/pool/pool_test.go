package pool_test

import (
	"testing"

	"github.com/Radon10043/cloud/src/pkg/pool"
)

var SPOOL = &pool.SpecPool{}

func init() {
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
