package osutil

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

type OsType int

const (
	unknown OsType = iota
	Linux
	FreeBSD
)

// String returns the string representation of OsType
func (o OsType) String() string {
	switch o {
	case Linux:
		return "linux"
	case FreeBSD:
		return "freebsd"
	default:
		return "unknown"
	}
}

// KernFilePath returns the expected path of kernel object file based on the OS type and given prefix
func (o OsType) KernFilePath(prefix string) string {
	switch o {
	case Linux:
		return filepath.Join(prefix, "vmlinux")
	case FreeBSD:
		return filepath.Join(prefix, "sys", runtime.GOARCH, "compile", "CLOUD", "kernel.full")
	default:
		return ""
	}
}

// parseOsType parses a string to osType, default to unknown if not matched
func ParseOsType(s string) (OsType, error) {
	switch strings.ToLower(s) {
	case "linux":
		return Linux, nil
	case "freebsd":
		return FreeBSD, nil
	default:
		return unknown, fmt.Errorf("unknown os type: %s", s)
	}
}
