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
	OpenBSD
	NetBSD
	Android
)

// String returns the string representation of OsType
func (o OsType) String() string {
	switch o {
	case Linux:
		return "linux"
	case FreeBSD:
		return "freebsd"
	case OpenBSD:
		return "openbsd"
	case NetBSD:
		return "netbsd"
	case Android:
		return "android"
	default:
		return "unknown"
	}
}

// KernExtractPath returns the expected path of kernel source for syz-extract based on the OS type and given prefix
func (o OsType) KernExtractPath(prefix string) string {
	switch o {
	case Android:
		return filepath.Join(prefix, "common")
	default:
		return prefix
	}
}

// KernFilePath returns the expected path of kernel object file based on the OS type and given prefix
func (o OsType) KernFilePath(prefix string) string {
	switch o {
	case Linux:
		return filepath.Join(prefix, "vmlinux")
	case FreeBSD:
		return filepath.Join(prefix, "sys", runtime.GOARCH, "compile", "SYZPILOT", "kernel.full")
	case OpenBSD:
		return filepath.Join(prefix, "sys", "arch", runtime.GOARCH, "compile", "SYZPILOT", "obj", "bsd.gdb")
	case NetBSD:
		return filepath.Join(prefix, "sys", "arch", runtime.GOARCH, "compile", "obj", "SYZPILOT", "netbsd.gdb")
	case Android:
		return filepath.Join(prefix, "dist", "vmlinux")
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
	case "openbsd":
		return OpenBSD, nil
	case "netbsd":
		return NetBSD, nil
	case "android":
		return Android, nil
	default:
		return unknown, fmt.Errorf("unknown os type: %s", s)
	}
}
