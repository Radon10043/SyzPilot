package tools

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Radon10043/cloud/src/generator/check"
)

type CheckSpecValidity struct {
	sc *check.SyzCheck
}

func NewCheckSpecValidityTool(sc *check.SyzCheck) *CheckSpecValidity {
	return &CheckSpecValidity{sc: sc}
}

func (t CheckSpecValidity) Name() string { return "check_spec_validity" }
func (t CheckSpecValidity) Description() string {
	return "Check the validity of syzlang specification, return error message if invalid."
}
func (t CheckSpecValidity) Call(ctx context.Context, input string) (string, error) {
	t.sc.CleanWorkdir()
	path := filepath.Join(t.sc.Workdir, "sys", "linux", "spec.txt")
	os.WriteFile(path, []byte(input), 0644)
	_, stderr, err := t.sc.ExtractConst()
	if err != nil {
		return stderr.String(), err
	}
	_, stderr, err = t.sc.CheckValidity()
	if err != nil {
		return stderr.String(), err
	}
	return "Specification is valid.", nil
}
