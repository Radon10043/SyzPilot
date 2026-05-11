package check

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	osu "github.com/Radon10043/SyzPilot/src/pkg/osutil"
	"github.com/Radon10043/SyzPilot/src/pkg/utils"
	"github.com/otiai10/copy"
)

type SpecCheck struct {
	Os               osu.OsType // target os type
	SyzExtract       string     // path to syz-extract binary
	SyzCheck         string     // path to syz-check binary
	KernelForExtract string     // path to kernel source for `make extract`
	KernelForCheck   string     // path to kernel source for syz-check
	Workdir          string     // working directory for syz-check, generally syzkaller's directory
	Sysdir           string     // path to sys directory (syzkaller/sys like structure)
}

type Option func(*SpecCheck)

// NewSpecCheck creates a new SpecCheck instance with given options
func NewSpecCheck(opts ...Option) *SpecCheck {
	osType, _ := osu.ParseOsType(runtime.GOOS)
	sc := &SpecCheck{
		Os:         osType,
		SyzExtract: "./bin/syz-extract",
		SyzCheck:   "./bin/syz-check",
		Sysdir:     "./syzkaller/sys",
	}
	for _, opt := range opts {
		opt(sc)
	}
	return sc
}

// WithOs sets the Os field of SpecCheck
func WithOs(osType osu.OsType) Option {
	return func(sc *SpecCheck) {
		sc.Os = osType
	}
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

// WithSysdir sets the Sysdir field of SpecCheck
func WithSysdir(path string) Option {
	return func(sc *SpecCheck) {
		sc.Sysdir = path
	}
}

// SetupWorkdir setup sc.Workdir by copy sc.Sysdir to sc.Workdir/
// TODO: It is very possible that new generated specs will conflict to existing specs.
// To solve such conflicts, there are two ways:
//  1. if current element existed in syzkaller, dont re-generate it.
//  2. create a minimal set of specs which will not impact spec generation. We can caluclate
//     dependency from sys/linux/sys.txt, and only copy the necessary specs to workdir.
func (sc *SpecCheck) SetupWorkdir() error {
	src := sc.Sysdir
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

// AddSpec add a syscall spec to sc.Workdir/sys/$TARGETOS/ as a temporary file.
// Return the path to spec file and error info.
func (sc *SpecCheck) AddSpec(spec string) (string, error) {
	sysDir := filepath.Join(sc.Workdir, "sys", sc.Os.String())
	file, err := os.CreateTemp(sysDir, "spec-*.txt")
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = file.WriteString(spec)
	if err != nil {
		return "", err
	}
	return file.Name(), nil
}

// ExtractConst run syz-extract to extract constants for a specific file (under sc.Workdir/sys/$OS/)
// from kernel source, return stdout and stderr of the command, also validity of the spec.
// If fname is empty, extract constants for all specs under sc.Workdir/sys/$OS/*.txt
func (sc *SpecCheck) ExtractConst(fname string) (*bytes.Buffer, *bytes.Buffer, bool) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command( // update command, only extract constants for one file
		sc.SyzExtract,
		"-build",
		"-arch="+runtime.GOARCH,
		"-sourcedir="+sc.KernelForExtract,
		"-os="+sc.Os.String(),
		fname,
	)
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Run() // it's okay to ignore error here since it is not fatal
	return sc.formatOutput(&stdout, &stderr)
}

// CheckValidity run syz-check to check validity of existing specs under sc.Workdir/sys/$OS/*.txt,
// return stdout and stderr of the command, also validity of the spec
func (sc *SpecCheck) CheckValidity() (*bytes.Buffer, *bytes.Buffer, bool) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(
		sc.SyzCheck,
		"-obj-"+runtime.GOARCH+"="+sc.Os.KernFilePath(sc.KernelForCheck),
		"-os="+sc.Os.String(),
	)
	cmd.Dir = sc.Workdir
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Run() // it's okay to ignore error here since it is not fatal
	return sc.formatOutput(&stdout, &stderr)
}

// formatOutput process the output of syz-extract or syz-check according to SpecCheck settings
func (sc *SpecCheck) formatOutput(stdout *bytes.Buffer, stderr *bytes.Buffer) (*bytes.Buffer, *bytes.Buffer, bool) {
	var (
		fStdout bytes.Buffer = *stdout
		fStderr bytes.Buffer = *stderr
	)
	fStdout = utils.RemoveLines(&fStdout, []string{"generating"}) // looks fishy
	fStderr = utils.RemoveLines(&fStderr, []string{"generating"})
	return &fStdout, &fStderr, len(fStdout.Bytes()) == 0 && len(fStderr.Bytes()) == 0
}
