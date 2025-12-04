package check

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-git/go-git/v6"
)

type SyzCheck struct {
	Bin              string // path to syz-check binary
	KernelForExtract string // path to kernel source for `make extract`
	KernelForCheck   string // path to kernel source for syz-check
	Workdir          string // working directory for syz-check, generally syzkaller's directory
}

// CheckWorkdir check whether sc.Workdir is syzkaller repository
func (sc *SyzCheck) CheckWorkdir() error {
	repo, err := git.PlainOpen(sc.Workdir)
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
		return fmt.Errorf("workdir is not syzkaller's directory")
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

// CleanWorkdir clean sc.Workdir
func (sc *SyzCheck) CleanWorkdir() error {
	repo, err := git.PlainOpen(sc.Workdir)
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

// AddSpec add a syscall spec to sc.Workdir/sys/linux/
func (sc *SyzCheck) AddSpec(spec string) error {
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
func (sc *SyzCheck) ExtractConst() (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(
		"make",
		"extract",
		"TARGETOS=linux",
		"SOURCEDIR="+sc.KernelForExtract,
		"SYZ_ENV=1", // prevent warning for compatibility
	)
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}

// CheckValidity run syz-check to check validity of existing specs under sc.Workdir/sys/$OS/*.txt,
// return stdout, stderr, and error of the command
func (sc *SyzCheck) CheckValidity() (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(sc.Bin, "-obj-amd64="+filepath.Join(sc.KernelForCheck, "vmlinux"))
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return stdout, stderr, err
	}
	return stdout, stderr, nil
}
