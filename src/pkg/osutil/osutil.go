package osutil

import (
	"fmt"
	"strings"
)

type OsType int

const (
	unknown OsType = iota
	Linux
	FreeBSD
)

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
