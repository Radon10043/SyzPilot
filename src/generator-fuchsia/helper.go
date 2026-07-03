package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	osu "github.com/Radon10043/cloud/src/pkg/osutil"
	"github.com/otiai10/copy"
)

// getcwd Gets current working directory
func getcwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic("getcwd() failed.")
	}
	return cwd
}

// defaultFixSysPrompt returns paths to the default fix system prompts
func defaultFixSysPrompt() string {
	dir := filepath.Join(getcwd(), "data", "prompts", "fix-fuchsia")
	return strings.Join([]string{
		filepath.Join(dir, "instruction.md"),
		filepath.Join(dir, "example_bluetooth.md"),
		filepath.Join(dir, "example_camera.md"),
	}, ",")
}

// duplicateKernel duplicate kernel directories/files for extract/check to the workdir,
// return extract kernel path, check kernel path, and error
func duplicateKernel(prefix string, kernel string, osType osu.OsType, logger *log.Logger) (string, string, error) {
	// duplicate kernel for extract
	logger.Printf("copying extract kernel to workdir (%s) ...\n", prefix)
	src := osType.KernExtractPath(kernel)
	dst := filepath.Join(prefix, "kernel-extract")
	if err := copy.Copy(src, dst); err != nil {
		logger.Printf("failed to copy extract kernel: %v\n", err)
		return "", "", err
	}

	// duplicate kernel for check
	logger.Printf("copying check kernel to workdir (%s) ...\n", prefix)
	src = osType.KernFilePath(kernel)
	dst = osType.KernFilePath(filepath.Join(prefix, "kernel-check"))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		logger.Printf("failed to create directory for check kernel: %v\n", err)
		return "", "", err
	}
	if err := copy.Copy(src, dst); err != nil {
		logger.Printf("failed to copy check kernel: %v\n", err)
		return "", "", err
	}
	return filepath.Join(prefix, "kernel-extract"), filepath.Join(prefix, "kernel-check"), nil
}

// dirsize calculates size of a specific directory, returns directory size (bytes) and error messages
func dirsize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		return nil
	})
	return size, err
}

// loadSysPrompt reads and concatenates the comma-separated prompt files into a single
// system prompt string.
func loadSysPrompt(paths string) (string, error) {
	var b strings.Builder
	for _, p := range strings.Split(paths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file %v: %v", p, err)
		}
		b.Write(data)
		b.WriteString("\n\n")
	}
	return b.String(), nil
}
