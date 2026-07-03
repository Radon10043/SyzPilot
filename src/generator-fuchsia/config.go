package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Radon10043/cloud/src/pkg/check"
	osu "github.com/Radon10043/cloud/src/pkg/osutil"
)

type ProgConfig struct {
	// agent configs
	Model string
	Env   string

	// kernel configs
	Outdir     string
	ExtractBin string
	CheckBin   string
	Kernel     string
	Resume     bool

	// spec generation configs
	Sysdir     string
	FidlsyzDir string

	// prompt configs
	FixSysPrompt string
	MaxFix       int

	// parallel configs
	Jobs int
}

// checkSysdir checks whther specs in Sysdir can be successfully passed syz-extract and syz-check,
// returns error messages.
func (pc *ProgConfig) checkSysdir() error {
	logger := log.New(os.Stdout, "[pre-check] ", log.LstdFlags|log.Lmsgprefix)
	workdir, err := os.MkdirTemp(os.TempDir(), "check-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary workdir: %v\n", err)
	}
	kernelExtract, kernelCheck, err := duplicateKernel(workdir, pc.Kernel, osu.Fuchsia, logger)
	sc := check.NewSpecCheck(
		check.WithOs(osu.Fuchsia),
		check.WithSyzExtract(pc.ExtractBin),
		check.WithSyzCheck(pc.CheckBin),
		check.WithKernelForExtract(kernelExtract),
		check.WithKernelForCheck(kernelCheck),
		check.WithWorkdir(workdir),
		check.WithSysdir(pc.Sysdir),
	)

	defer os.RemoveAll(workdir)
	if err := sc.SetupWorkdir(); err != nil {
		return fmt.Errorf("failed to setup workdir: %v\n", err)
	}

	if stdout, stderr, valid := sc.ExtractConst(""); !valid {
		return fmt.Errorf("failed to execute syz-extract with sysdir:\n\nstdout:\n%v\n\nstderr:\n%v\n", stdout, stderr)
	}
	if stdout, stderr, valid := sc.CheckValidity(); !valid {
		return fmt.Errorf("failed to execute syz-check with sysdir:\n\nstdout:\n%v\n\nstderr:\n%v\n", stdout, stderr)
	}
	logger.Printf("%v is valid!", pc.Sysdir)
	return nil
}

// checkValid checks validity of provided configs, returns error if the provided configs
// is considered as invalid.
func (pc *ProgConfig) checkValid() error {
	// check file/dir existence
	type safe struct {
		err  error
		stat func(file string, title string)
	}
	s := safe{err: nil}
	s.stat = func(file string, title string) {
		if s.err != nil {
			return
		}
		_, err := os.Stat(file)
		if err != nil {
			s.err = fmt.Errorf("%v: %v", title, err)
		}
	}
	s.stat(pc.Env, "-env")
	s.stat(pc.ExtractBin, "-extract-bin")
	s.stat(pc.CheckBin, "-check-bin")
	s.stat(pc.Kernel, "-kernel")
	s.stat(pc.Sysdir, "-sysdir")
	s.stat(pc.FidlsyzDir, "-fidlsyz-dir")
	if s.err != nil {
		return s.err
	}

	// other checks
	if pc.MaxFix < 0 {
		return fmt.Errorf("-max-fix: must be 0 or positive.\n")
	}
	if pc.Jobs < 1 {
		return fmt.Errorf("-jobs: must be positive.\n")
	}
	return nil
}

// checkEmpty check whether fields that not allowed empty are set empty, these fields are
// most of them points to a path. Returns error if a field is empty unexpected.
func (pc *ProgConfig) checkEmpty() error {
	var err error
	empty := func(value string, name string) {
		if err != nil {
			return
		}
		if value == "" {
			err = fmt.Errorf("%s shouldn't be empty.", name)
		}
	}
	empty(pc.Model, "-model")
	empty(pc.Env, "-env")
	empty(pc.Outdir, "-outdir")
	empty(pc.ExtractBin, "-extract-bin")
	empty(pc.CheckBin, "-check-bin")
	empty(pc.Kernel, "-kernel")
	empty(pc.FixSysPrompt, "-fix-sys-prompt")
	empty(pc.Sysdir, "-sysdir")
	empty(pc.FidlsyzDir, "-fidlsyz-dir")
	return err
}

// toAbs converts related fields in ProgConfig to corresponding absolute path, returns
// error if failed to convert to absolute path
func (pc *ProgConfig) toAbs() error {
	type safe struct {
		err error
		abs func(path string) string
	}
	s := safe{err: nil}
	s.abs = func(path string) string {
		if s.err != nil {
			return path
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			s.err = fmt.Errorf("failed to abs: %v\n", err)
			return path
		}
		return absPath
	}
	pc.Env = s.abs(pc.Env)
	pc.Outdir = s.abs(pc.Outdir)
	pc.ExtractBin = s.abs(pc.ExtractBin)
	pc.CheckBin = s.abs(pc.CheckBin)
	pc.Kernel = s.abs(pc.Kernel)
	pc.Sysdir = s.abs(pc.Sysdir)
	pc.FidlsyzDir = s.abs(pc.FidlsyzDir)
	var absFixSysPrompt strings.Builder
	for f := range strings.SplitSeq(pc.FixSysPrompt, ",") {
		absFixSysPrompt.WriteString(s.abs(f))
		absFixSysPrompt.WriteString(",")
	}
	pc.FixSysPrompt = strings.TrimRight(absFixSysPrompt.String(), ",")
	return s.err
}
