package osutil

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Radon10043/cloud/src/pkg/tools"
	"github.com/Radon10043/cloud/src/pkg/tools/csrc"
	"github.com/Radon10043/cloud/src/pkg/tools/fuchsia"
)

type OsType int

const (
	unknown OsType = iota
	Linux
	FreeBSD
	OpenBSD
	NetBSD
	Android
	Fuchsia
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
	case Fuchsia:
		return "fuchsia"
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
		return filepath.Join(prefix, "sys", runtime.GOARCH, "compile", "CLOUD", "kernel.full")
	case OpenBSD:
		return filepath.Join(prefix, "sys", "arch", runtime.GOARCH, "compile", "CLOUD", "obj", "bsd.gdb")
	case NetBSD:
		return filepath.Join(prefix, "sys", "arch", runtime.GOARCH, "compile", "obj", "CLOUD", "netbsd.gdb")
	case Android:
		return filepath.Join(prefix, "dist", "vmlinux")
	case Fuchsia:
		fuchsiaArch := "x64"
		if runtime.GOARCH != "amd64" {
			fuchsiaArch = runtime.GOARCH
		}
		return filepath.Join(prefix, "out", fuchsiaArch, "kernel_"+fuchsiaArch+"-kasan-sancov", "vmzircon")
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
	case "fuchsia":
		return Fuchsia, nil
	default:
		return unknown, fmt.Errorf("unknown os type: %s", s)
	}
}

// Distclean performs OS-specific distclean operation in the given directory
func (o OsType) Distclean(dir string) (*bytes.Buffer, *bytes.Buffer, error) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
		cmd    *exec.Cmd
	)
	switch o {
	case Linux, Android:
		cmd = exec.Command("make", "distclean")
	default: // no need to run distclean?
		return nil, nil, nil
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return &stdout, &stderr, err
}

// AgentToolMap returns the tool map for the agent based on the OS type
func (o OsType) AgentToolMap() (map[string]tools.ToolExec, error) {
	// c source tools
	csrcToolMap := map[string]tools.ToolExec{
		csrc.GetFuncCodeByNameTool.Function.Name:         {Tool: csrc.GetFuncCodeByNameTool, Exec: csrc.ExecGetFuncCodeByName},
		csrc.GetEnumCodeByEnumeratorTool.Function.Name:   {Tool: csrc.GetEnumCodeByEnumeratorTool, Exec: csrc.ExecGetEnumCodeByEnumerator},
		csrc.GetEnumCodeBySpecifierTool.Function.Name:    {Tool: csrc.GetEnumCodeBySpecifierTool, Exec: csrc.ExecGetEnumCodeBySpecifier},
		csrc.GetStructCodeByNameTool.Function.Name:       {Tool: csrc.GetStructCodeByNameTool, Exec: csrc.ExecGetStructCodeByName},
		csrc.GetUnionCodeByNameTool.Function.Name:        {Tool: csrc.GetUnionCodeByNameTool, Exec: csrc.ExecGetUnionCodeByName},
		csrc.GetGlobalVarCodeByNameTool.Function.Name:    {Tool: csrc.GetGlobalVarCodeByNameTool, Exec: csrc.ExecGetGlobalVarCodeByName},
		csrc.GetTypedefCodeByDefineTool.Function.Name:    {Tool: csrc.GetTypedefCodeByDefineTool, Exec: csrc.ExecGetTypedefCodeByDefine},
		csrc.GetTypedefTypeByDefineTool.Function.Name:    {Tool: csrc.GetTypedefTypeByDefineTool, Exec: csrc.ExecGetTypedefTypeByDefine},
		csrc.GetMacroDefCodeByNameTool.Function.Name:     {Tool: csrc.GetMacroDefCodeByNameTool, Exec: csrc.ExecGetMacroDefCodeByName},
		csrc.GetMacroDefCodesByPatternTool.Function.Name: {Tool: csrc.GetMacroDefCodesByPatternTool, Exec: csrc.ExecGetMacroDefCodesByPattern},
		csrc.GetMacroDefLocByNameTool.Function.Name:      {Tool: csrc.GetMacroDefLocByNameTool, Exec: csrc.ExecGetMacroDefLocByName},
	}

	// fuchsia tools
	fuchsiaToolMap := map[string]tools.ToolExec{
		fuchsia.GetDeclByNameTool.Function.Name:            {Tool: fuchsia.GetDeclByNameTool, Exec: fuchsia.ExecGetDeclByName},
		fuchsia.GetAliasTypeByNameTool.Function.Name:       {Tool: fuchsia.GetAliasTypeByNameTool, Exec: fuchsia.ExecGetAliasTypeByName},
		fuchsia.GetBitsMembersByNameTool.Function.Name:     {Tool: fuchsia.GetBitsMembersByNameTool, Exec: fuchsia.ExecGetBitsMembersByName},
		fuchsia.GetBitsTypeByNameTool.Function.Name:        {Tool: fuchsia.GetBitsTypeByNameTool, Exec: fuchsia.ExecGetBitsTypeByName},
		fuchsia.GetConstTypeByNameTool.Function.Name:       {Tool: fuchsia.GetConstTypeByNameTool, Exec: fuchsia.ExecGetConstTypeByName},
		fuchsia.GetConstValueByNameTool.Function.Name:      {Tool: fuchsia.GetConstValueByNameTool, Exec: fuchsia.ExecGetConstValueByName},
		fuchsia.GetEnumTypeByNameTool.Function.Name:        {Tool: fuchsia.GetEnumTypeByNameTool, Exec: fuchsia.ExecGetEnumTypeByName},
		fuchsia.GetEnumMembersByNameTool.Function.Name:     {Tool: fuchsia.GetEnumMembersByNameTool, Exec: fuchsia.ExecGetEnumMembersByName},
		fuchsia.GetProtocolMethodsByNameTool.Function.Name: {Tool: fuchsia.GetProtocolMethodsByNameTool, Exec: fuchsia.ExecGetProtocolMethodsByName},
		fuchsia.GetProtocolAttrsByNameTool.Function.Name:   {Tool: fuchsia.GetProtocolAttrsByNameTool, Exec: fuchsia.ExecGetProtocolAttrsByName},
		fuchsia.GetServiceMembersByNameTool.Function.Name:  {Tool: fuchsia.GetServiceMembersByNameTool, Exec: fuchsia.ExecGetServiceMembersByName},
		fuchsia.GetStructMembersByNameTool.Function.Name:   {Tool: fuchsia.GetStructMembersByNameTool, Exec: fuchsia.ExecGetStructMembersByName},
		fuchsia.GetStructTypeShapeByNameTool.Function.Name: {Tool: fuchsia.GetStructTypeShapeByNameTool, Exec: fuchsia.ExecGetStructTypeShapeByName},
		fuchsia.GetTableMembersByNameTool.Function.Name:    {Tool: fuchsia.GetTableMembersByNameTool, Exec: fuchsia.ExecGetTableMembersByName},
		fuchsia.GetTableTypeShapeByNameTool.Function.Name:  {Tool: fuchsia.GetTableTypeShapeByNameTool, Exec: fuchsia.ExecGetTableTypeShapeByName},
		fuchsia.GetUnionMembersByNameTool.Function.Name:    {Tool: fuchsia.GetUnionMembersByNameTool, Exec: fuchsia.ExecGetUnionMembersByName},
		fuchsia.GetUnionTypeShapeByNameTool.Function.Name:  {Tool: fuchsia.GetUnionTypeShapeByNameTool, Exec: fuchsia.ExecGetUnionTypeShapeByName},
	}

	switch o {
	case Linux, Android, FreeBSD, OpenBSD, NetBSD:
		return csrcToolMap, nil
	case Fuchsia:
		return fuchsiaToolMap, nil
	}

	return nil, fmt.Errorf("no appropriate tools for %v.", o.String())
}
