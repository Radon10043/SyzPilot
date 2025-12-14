package check

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-git/go-git/v6"
	"github.com/otiai10/copy"
)

type SpecCheck struct {
	SyzExtract       string // path to syz-extract binary
	SyzCheck         string // path to syz-check binary
	KernelForExtract string // path to kernel source for `make extract`
	KernelForCheck   string // path to kernel source for syz-check
	Workdir          string // working directory for syz-check, generally syzkaller's directory
	Syzkaller        string // path to syzkaller repository
}

type Option func(*SpecCheck)

// NewSpecCheck creates a new SpecCheck instance with given options
func NewSpecCheck(opts ...Option) *SpecCheck {
	sc := &SpecCheck{
		SyzExtract: "./bin/syz-extract",
		SyzCheck:   "./bin/syz-check",
		Syzkaller:  "./syzkaller",
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

// WithSyzExtract sets the SyzExtract field of SpecCheck
func WithSyzExtract(path string) Option {
	return func(sc *SpecCheck) {
		sc.SyzExtract = path
	}
}

// WithSyzCheck sets the SyzCheck field of SpecCheck
func WithSyzCheck(path string) Option {
	return func(sc *SpecCheck) {
		sc.SyzCheck = path
	}
}

// WithKernelForExtract sets the KernelForExtract field of SpecCheck
func WithKernelForExtract(path string) Option {
	return func(sc *SpecCheck) {
		sc.KernelForExtract = path
	}
}

// WithKernelForCheck sets the KernelForCheck field of SpecCheck
func WithKernelForCheck(path string) Option {
	return func(sc *SpecCheck) {
		sc.KernelForCheck = path
	}
}

// WithWorkdir sets the Workdir field of SpecCheck
func WithWorkdir(path string) Option {
	return func(sc *SpecCheck) {
		sc.Workdir = path
	}
}

// WithSyzkaller sets the Syzkaller field of SpecCheck
func WithSyzkaller(path string) Option {
	return func(sc *SpecCheck) {
		sc.Syzkaller = path
	}
}

// CheckSyzkaller check whether path to syzkaller repository is valid
func (sc *SpecCheck) CheckSyzkaller() error {
	repo, err := git.PlainOpen(sc.Syzkaller)
	if err != nil {
		return err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return err
	}
	isSyzkaller := false
	for _, remote := range remotes {
		urls := remote.Config().URLs
		if urls[0] == "https://github.com/google/syzkaller" {
			isSyzkaller = true
			break
		}
	}
	if !isSyzkaller {
		return fmt.Errorf("not syzkaller repository: %s", sc.Syzkaller)
	}
	head, err := repo.Head()
	if err != nil {
		return err
	}
	if head.Hash().String()[:8] != "4b25d554" {
		log.Println("I use syzkaller 4b25d554 btw :)")
	}
	return nil
}

// CleanSyzkaller clean sc.Syzkaller repository to its original state
func (sc *SpecCheck) CleanSyzkaller() error {
	repo, err := git.PlainOpen(sc.Syzkaller)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Clean(&git.CleanOptions{Dir: true})
	if err != nil {
		return err
	}
	err = w.Reset(&git.ResetOptions{Mode: git.HardReset})
	if err != nil {
		return err
	}
	return nil
}

// SetupWorkdir setup sc.Workdir by copy sc.Syzkaller/sys/linux/* to sc.Workdir/sys/linux/
func (sc *SpecCheck) SetupWorkdir() error {
	src := filepath.Join(sc.Syzkaller, "sys")
	dst := filepath.Join(sc.Workdir, "sys")
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return fmt.Errorf("failed to create sys in workdir: %v", err)
	}
	return copy.Copy(src, dst)
}

// CleanWorkdir remove sc.Workdir/sys
func (sc *SpecCheck) CleanWorkdir() error {
	if err := os.RemoveAll(filepath.Join(sc.Workdir, "sys")); err != nil {
		return fmt.Errorf("failed to remove sys in workdir: %v", err)
	}
	return nil
}

// RestoreWorkdir clean and setup sc.Workdir
func (sc *SpecCheck) RestoreWorkdir() error {
	if err := sc.CleanWorkdir(); err != nil {
		return fmt.Errorf("failed to clean workdir: %v", err)
	}
	return sc.SetupWorkdir()
}

// AddSpec add a syscall spec to sc.Workdir/sys/linux/
func (sc *SpecCheck) AddSpec(spec string) error {
	sysDir := filepath.Join(sc.Workdir, "sys", "linux")
	file, err := os.CreateTemp(sysDir, "spec-*.txt")
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(spec)
	if err != nil {
		return err
	}
	return nil
}

// ExtractConst run `make extract` to extract constants from kernel source, return stdout, stderr, and error of the command
func (sc *SpecCheck) ExtractConst() (*bytes.Buffer, *bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(
		sc.SyzExtract,
		"-build",
		"-arch="+runtime.GOARCH,
		"-sourcedir="+sc.KernelForExtract,
		"-os=linux",
	)
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return &stdout, &stderr, err
	}
	return &stdout, &stderr, nil
}

// CheckValidity run syz-check to check validity of existing specs under sc.Workdir/sys/$OS/*.txt,
// return stdout, stderr, and error of the command
func (sc *SpecCheck) CheckValidity() (*bytes.Buffer, *bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sc.SyzCheck, "-obj-amd64="+filepath.Join(sc.KernelForCheck, "vmlinux"))
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return &stdout, &stderr, err
	}
	return &stdout, &stderr, nil
}
